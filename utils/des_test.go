package utils

import (
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDesEncrypt(t *testing.T) {
	r := require.New(t)
	//data, err := TripleEcbDesEncrypt([]byte(`cagent=81288128/\\\\/method=tc`), []byte("12341234"))
	//r.NoError(err)
	//fmt.Println(base64.StdEncoding.EncodeToString(data))
	d, err := base64.StdEncoding.DecodeString("gNIOo8HUnuyQOgsP9g29f1P6vScuNwtfoadgWAADPsayU9ATcK3RuTNDr82af6JAJng3+262TzGUlf+N2Z2p3cW0LYa/lAGINSGeQ6Sv6omZkAftwGW15moQAIrWlv+gJ8QkaDdW8tAOIxa1W+SNJrO7IOIS+FFJ4mC9Ncl6Hwz03eLBU3DrIdWDFDrxHWHaKnhL/64KO4jLQqaglV1JS9Fuxj5cM3dM/fdRZQGdlCfkuPCx9dwybQ==")
	r.NoError(err)
	data, err := TripleEcbDesDecrypt(d, []byte("df7pcDRK"))
	r.NoError(err)
	fmt.Println(string(data))
	//r.Equal([]byte(`cagent=81288128/\\\\/method=tc`), data)
}
