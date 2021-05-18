package main

import (
	"errors"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"

	"github.com/FJSDS/common/eventloop"
	"github.com/FJSDS/common/logger"
	"github.com/FJSDS/common/network/basepb"
	"github.com/FJSDS/common/network/tcp"
	"github.com/FJSDS/common/ulimit"
)

type Client struct {
	*tcp.Connector
	log *logger.Logger
}

func (this_ *Client) OnSessionConnected(s *tcp.Session) {
	//this_.TestRequest()
}

func (this_ *Client) TestRequest() {
	this_.Connector.RequestNoError(&basepb.Base{}, func(m proto.Message) {
		msg, ok := m.(*basepb.Base_Success)
		if ok {
			this_.log.Info("recv response", zap.String("msgID", proto.MessageName(m)), zap.String("msg", msg.String()))
			this_.TestRequest()
		}
	})
}

func (this_ *Client) OnSessionDisConnected(s *tcp.Session, e error) {
	this_.log.Info("connect failed", zap.Error(e))
}

func (this_ *Client) OnRPCRequest(s *tcp.Session, msg proto.Message) proto.Message {
	return &basepb.Base_Success{}
}

func (this_ *Client) OnNormalMsg(s *tcp.Session, msg proto.Message) {

}

func main() {
	log, err := logger.NewLogger("tcp_testclient", ".", zap.InfoLevel)
	if err != nil {
		panic(err)
	}
	err = ulimit.SetRLimit()
	if err != nil {
		panic(err)
	}

	queue := eventloop.NewEventLoop(log)
	queue.Start(tcp.DispatchMsg(func(event interface{}) {
		switch e := event.(type) {
		default:
			log.Warn("unknown event", zap.Any("event", e))
		}
	}), func() {
		log.Info("event queue stopped")
	})
	clients := make([]*Client, 0, 100)
	for i := 0; i < 30000; i++ {
		clients = append(clients, startOne(queue, log))
		//time.Sleep(time.Second * 10)
		time.Sleep(time.Microsecond * 100)
	}
	fmt.Println(111111111111)
	go func() {
		err := http.ListenAndServe("0.0.0.0:8880", nil)
		fmt.Println(err)
	}()

	sigShutdown := make(chan os.Signal, 5)
	signal.Notify(sigShutdown, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	//signal.Notify(sigShutdown)
	sigCall := <-sigShutdown
	log.Info("recv signal", zap.String("signal", sigCall.String()))
	for _, v := range clients {
		v.Close(errors.New("shutdown"))
	}
	queue.Stop()
}

func startOne(queue *eventloop.EventLoop, log *logger.Logger) *Client {
	connector := tcp.NewConnector("192.168.1.62:9999", queue, log, "test_client", tcp.WithOpenDebugLog(), tcp.WithReconnect())
	c := &Client{
		Connector: connector,
		log:       log,
	}
	connector.SetCallback(c)
	connector.Connect()
	return c
}
