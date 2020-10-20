package consumeserver

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	consulclient "git.esports.com/ddm/common/consul_client"
	"git.esports.com/ddm/common/db"
	"git.esports.com/ddm/common/logger"
	"git.esports.com/ddm/common/network/ulimit"
	workpool "git.esports.com/ddm/common/work_pool"
	goprotovalidators "github.com/FJSDS/go-proto-validators"
	"github.com/golang/protobuf/proto"
	"github.com/jmoiron/sqlx"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"

	"git.esports.com/ddm/DouZhe/model/base"
	"git.esports.com/ddm/DouZhe/natsclient"
	"git.esports.com/ddm/DouZhe/pbmsg"
)

type ConsumeServer struct {
	nc         *natsclient.NATSClient
	ch         chan *nats.Msg
	ss         *nats.Subscription
	sss        []*nats.Subscription
	workPool   *workpool.WorkPool
	log        *logger.Logger
	doc        map[string]*base.DocInfo
	routerDB   *db.MySQL
	HandleMap  map[string]func(*base.Request) *base.Response
	version    string
	serverName string
	closed     chan struct{}
	isDebug    bool
}

func NewConsumeServer(log *logger.Logger, consulAddr string, version string, serverName string) (*ConsumeServer, error) {
	consulClient, err := consulclient.NewConsulClient(consulAddr, "ConsumeServer", log)
	if err != nil {
		return nil, err
	}
	natsAddrBytes, err := consulClient.GetKey("nats.addr")
	if err != nil {
		return nil, err
	}
	var natsAddrS []string
	err = json.Unmarshal(natsAddrBytes, &natsAddrS)
	if err != nil {
		return nil, err
	}
	nc, err := natsclient.NewNATSClient(log, natsAddrS)
	if err != nil {
		return nil, err
	}
	routerDBConfig, err := consulClient.GetMysql("mysql.router")
	if err != nil {
		return nil, err
	}
	routerDBConfig.MaxIdleConnS = 0
	routerDB := db.NewMySQL(routerDBConfig, log)
	s := &ConsumeServer{
		nc:         nc,
		ch:         make(chan *nats.Msg, 10000),
		workPool:   workpool.NewWorkPool(0, log),
		log:        log,
		doc:        map[string]*base.DocInfo{},
		routerDB:   routerDB,
		HandleMap:  map[string]func(*base.Request) *base.Response{},
		version:    version,
		serverName: serverName,
		closed:     make(chan struct{}),
		isDebug:    log.IsLogDebug(),
	}
	return s, nil
}

func (this_ *ConsumeServer) GetWorkPool() *workpool.WorkPool {
	return this_.workPool
}
func (this_ *ConsumeServer) StoreHandler() error {
	return this_.routerDB.BeginTx(func(tx *sqlx.Tx) error {
		for _, v := range this_.doc {
			canType, _ := json.Marshal(v.CanType)
			_, err := tx.Exec("INSERT INTO router.handler(full_path,version,server_name, request_param_full_path, request_go_type,"+
				"response_param_full_path, response_go_type, can_type, `desc`, detail_desc,need_auth,only_server) "+
				"VALUES (?,?,?,?,?,?,?,?,?,?,?,?) ON DUPLICATE KEY UPDATE request_param_full_path=?,request_go_type=?,"+
				"response_param_full_path=?,response_go_type=?,can_type=?,`desc`=?,detail_desc=?,need_auth=?,only_server=?",
				v.FullPath, this_.version, this_.serverName, v.RequestParamFullPath, v.RequestGoType, v.ResponseParamFullPath, v.ResponseGoType, canType,
				v.Desc, v.DetailDesc, v.NeedAuth, v.OnlyServer, v.RequestParamFullPath, v.RequestGoType, v.ResponseParamFullPath, v.ResponseGoType, canType,
				v.Desc, v.DetailDesc, v.NeedAuth, v.OnlyServer)
			if err != nil {
				return err
			}
		}
		_, err := tx.Exec("INSERT INTO router.version(type, version,updated_time) VALUES (?,?,NOW()) ON DUPLICATE KEY UPDATE version=?,updated_time=NOW()", this_.serverName, this_.version, this_.version)
		return err
	})
}

func (this_ *ConsumeServer) Run(subj string, subs ...string) error {
	var err error
	this_.ss, err = this_.nc.ChanQueueSubscribe(subj, strings.Replace(subj, ".", "_", -1)+"_group", this_.ch)
	if err != nil {
		return err
	}
	for _, v := range subs {
		ss, err := this_.nc.ChanQueueSubscribe(v, strings.Replace(v, ".", "_", -1)+"_group", this_.ch)
		if err != nil {
			return err
		}
		this_.sss = append(this_.sss, ss)
	}
	err = pbmsg.StoreMysql(this_.routerDB)
	if err != nil {
		return err
	}
	err = this_.StoreHandler()
	if err != nil {
		return err
	}
	err = ulimit.SetRLimit()
	if err != nil {
		return err
	}
	this_.workPool.Run(func(i interface{}) {
		this_.log.Error("workPool panic", zap.Any("info", i))
	})
	go func() {
		this_.log.Info("ConsumeServer Start Consume", zap.String("subj", subj), zap.Any("subs", subs))
		for v := range this_.ch {
			msg := v
			this_.workPool.Post(func() {
				this_.ProcessOneMsg(msg)
			})
		}
		close(this_.closed)
	}()
	return nil
}

func (this_ *ConsumeServer) Shutdown() error {
	err := this_.ss.Drain()
	if err != nil {
		return err
	}
	this_.log.Info("this_.ss Drain")
	for this_.ss.IsValid() {
		time.Sleep(time.Millisecond * 10)
	}
	this_.log.Info("this_.ss end")
	for _, v := range this_.sss {
		err := v.Drain()
		if err != nil {
			return err
		}
		this_.log.Info("drain", zap.String("subject", v.Subject))
	}
	for _, v := range this_.sss {
		for v.IsValid() {
			time.Sleep(time.Millisecond * 10)
		}
		this_.log.Info("end", zap.String("subject", v.Subject))
	}
	close(this_.ch)
	this_.nc.Close()
	this_.log.Info("start close")
	<-this_.closed
	this_.log.Info("start closed")
	this_.workPool.Stop()
	this_.log.Info("start closed")
	return nil
}
func (this_ *ConsumeServer) GetNatsClient() *natsclient.NATSClient {
	return this_.nc
}

func (this_ *ConsumeServer) ProcessOneMsg(msg *nats.Msg) {
	r := &base.Request{}
	err := json.Unmarshal(msg.Data, r)
	if err != nil {
		data, _ := json.Marshal(&base.Response{
			Code:    int64(pbmsg.ErrorCode_CodeInvalidRequestData),
			Message: err.Error(),
		})
		_ = msg.Respond(data)
		return
	}
	handler, ok := this_.HandleMap[r.Path]
	if ok {
		resp := handler(r)
		data, err := json.Marshal(resp)
		if err != nil {
			data, _ = json.Marshal(&base.Response{
				Code:    int64(pbmsg.ErrorCode_CodeInvalidRequestData),
				Message: err.Error(),
			})
		}
		_ = msg.Respond(data)
		return
	}
	data, _ := json.Marshal(&base.Response{
		Code:    int64(pbmsg.ErrorCode_CodeAPINotFound),
		Message: "API接口未找到或已弃用",
	})
	_ = msg.Respond(data)
}

type RouteConfig struct {
	OnlyServer   int64
	CallRequest  func(req interface{})
	CallResponse func(req interface{})
}

type Option func(*RouteConfig) *RouteConfig

func OnlyServer(old *RouteConfig) *RouteConfig {
	if old == nil {
		old = &RouteConfig{}
	}
	old.OnlyServer = 1
	return old
}

func (this_ *ConsumeServer) Route(handler interface{}, desc string, detailDesc string, needAuth int64, opts ...Option) {
	fv := reflect.ValueOf(handler)
	if fv.Kind() != reflect.Func {
		this_.log.Fatal("Route() handler must be a func.", zap.String("handler", fv.String()))
	}

	ft := fv.Type()
	reqType1 := ft.In(0).String()
	if ft.NumIn() < 2 || reqType1 != "*base.Request" {
		this_.log.Fatal("RegisterHandler() handler function should be like func(info *base.Request, req *pbmsg.Account_Login) (resp*pbmsg.Account_Register,errorInfo*pbmsg.ErrorInfo)",
			zap.String("in handler", fv.String()))
	}
	param1Type := ft.In(1)
	if param1Type.Kind() != reflect.Ptr {
		this_.log.Fatal("RegisterHandler() handler function should be like func(info *base.Request, req *pbmsg.Account_Login) (resp*pbmsg.Account_Register,errorInfo*pbmsg.ErrorInfo)",
			zap.String("in handler", fv.String()))
	}

	reqType := param1Type.String()
	ty := param1Type.Elem()
	pbv := reflect.New(ty)
	msgIn, ok := pbv.Interface().(proto.Message)
	if !ok || msgIn == nil {
		this_.log.Fatal("RegisterHandler() handler function should be like func(info *base.Request, req *pbmsg.Account_Login) (resp*pbmsg.Account_Register,errorInfo*pbmsg.ErrorInfo)",
			zap.String("in handler", fv.String()))
	}
	msgInFullPath := proto.MessageName(msgIn)
	if msgInFullPath == "" {
		this_.log.Fatal("RegisterHandler() handler function should be like func(info *base.Request, req *pbmsg.Account_Login) (resp*pbmsg.Account_Register,errorInfo*pbmsg.ErrorInfo),req must in proto",
			zap.String("in handler", fv.String()))
	}

	if ft.NumOut() < 2 || ft.Out(1).String() != "*pbmsg.ErrorInfo" {
		this_.log.Fatal("RegisterHandler() handler function should be like func(info *base.Request, req *pbmsg.Account_Login) (resp*pbmsg.Account_Register,errorInfo*pbmsg.ErrorInfo)",
			zap.String("in handler", fv.String()))
	}

	out0Type := ft.Out(0)
	if out0Type.Kind() != reflect.Ptr {
		this_.log.Fatal("RegisterHandler() handler function should be like func(info *base.Request, req *pbmsg.Account_Login) (resp*pbmsg.Account_Register,errorInfo*pbmsg.ErrorInfo)",
			zap.String("in handler", fv.String()))
	}
	respType := out0Type.String()
	oty := out0Type.Elem()
	pbo := reflect.New(oty)
	msgOut, ok := pbo.Interface().(proto.Message)
	if !ok || msgOut == nil {
		this_.log.Fatal("RegisterHandler() handler function should be like func(info *base.Request, req *pbmsg.Account_Login) (resp*pbmsg.Account_Register,errorInfo*pbmsg.ErrorInfo)",
			zap.String("in handler", fv.String()))
	}
	msgOutFullPath := proto.MessageName(msgOut)
	if msgOutFullPath == "" {
		this_.log.Fatal("RegisterHandler() handler function should be like func(info *base.Request, req *pbmsg.Account_Login) (resp*pbmsg.Account_Register,errorInfo*pbmsg.ErrorInfo),resp must in proto",
			zap.String("in handler", fv.String()))
	}
	value := &base.DocInfo{
		Desc:                  desc,
		DetailDesc:            detailDesc,
		FullPath:              msgInFullPath,
		RequestParamFullPath:  msgInFullPath,
		RequestGoType:         reqType,
		ResponseParamFullPath: msgOutFullPath,
		ResponseGoType:        respType,
		CanType:               []string{"http"},
		NeedAuth:              needAuth,
	}
	rc := &RouteConfig{}
	for _, v := range opts {
		rc = v(rc)
	}
	value.OnlyServer = rc.OnlyServer
	if _, ok := this_.doc[msgInFullPath]; ok {
		this_.log.Fatal("handler already registered", zap.String("in handler", fv.String()))
	} else {
		this_.doc[msgInFullPath] = value
	}

	this_.log.Debug("register route", zap.String("uri", msgInFullPath), zap.String("func name", fv.String()))

	this_.HandleMap[msgInFullPath] = func(r *base.Request) (resp *base.Response) {
		defer func() {
			e := recover()
			if e == nil {
				if resp != nil && resp.Code != 0 {
					this_.log.Error("handler failed", zap.Any("req", r), zap.Any("resp", resp))
				} else {
					if this_.isDebug {
						this_.log.Debug("handler success", zap.Any("req", r), zap.Any("resp", resp))
					}
				}
			} else {
				this_.log.Error("handler panic", zap.Any("panic info", e), zap.Any("req", r), zap.Any("resp", resp))
				resp = &base.Response{
					Code:    int64(pbmsg.ErrorCode_CodeInternalServerError),
					Display: "server panic",
					Message: fmt.Sprintf("%v", e),
				}
			}
		}()
		tyd := reflect.New(ty)
		req := tyd.Interface().(proto.Message)
		err := json.Unmarshal(r.Body, req)
		if err != nil {
			reply := &base.Response{}
			reply.Code = int64(pbmsg.ErrorCode_CodeInvalidRequestData)
			reply.Display = "无效的请求参数"
			reply.Message = err.Error()
			return reply
		}
		if validator, ok := req.(interface {
			Validate() error
		}); ok {
			err = validator.Validate()
			if err != nil {
				reply := &base.Response{}
				e, ok := err.(*goprotovalidators.PbFieldError)
				if ok {
					reply.Fields = append(reply.Fields, e.FieldStack...)
				}
				reply.Message = err.Error()
				reply.Code = int64(pbmsg.ErrorCode_CodeInvalidRequestData)
				reply.Display = "无效的请求参数"
				return reply
			}
		}
		if rc.CallRequest != nil {
			rc.CallRequest(tyd.Elem().Addr().Interface())
		}
		out := fv.Call([]reflect.Value{reflect.ValueOf(r), tyd})
		reply := &base.Response{}
		if ei := out[1].Interface().(*pbmsg.ErrorInfo); ei == nil {
			if rc.CallResponse != nil {
				rc.CallResponse(out[0].Interface())
			}
			reply.Data = out[0].Interface()
			reply.Display = "请求成功"
			reply.Message = "success"
		} else {
			reply.Code = int64(ei.Code)
			reply.Message = ei.Message
			reply.Fields = ei.Fields
			reply.Display = ei.Display
		}
		return reply
	}
}
