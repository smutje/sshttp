package sshttp

import (
  "testing"
  //"net"
  "runtime/debug"
  "io"
  "net/http"
  "crypto/elliptic"
  "crypto/ecdsa"
  "crypto/rand"
  "code.google.com/p/gosshnew/ssh"
)

const(
  tmpSock = "/tmp/sshttp.sock"
)

func genKey() ssh.Signer {
  key, err := ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
  if err != nil {
    panic(err)
  }
  sig, err := ssh.NewSignerFromKey(key)
  if err != nil {
    panic(err)
  }
  return sig
}

var (
  sKey ssh.Signer
  cKey ssh.Signer
)

func init(){
  sKey = genKey()
  cKey = genKey()
}

func TestSimpleHttp(t *testing.T){
  conf := ssh.ServerConfig{
    PublicKeyCallback: AcceptPublicKey(cKey.PublicKey()),
  }
  conf.AddHostKey(sKey)
  /*l,err := net.Listen("unix",tmpSock)
  if err != nil {
    t.Fatal(err)
  }*/
  l := Listen()
  srv := Server{
    Handler: http.HandlerFunc(func(wr http.ResponseWriter,req *http.Request){
      wr.WriteHeader(204)
    }),
    ErrorHandler: func(err error){
      if err == io.EOF {
        t.Log("Ignoring EOF")
      }else{
        debug.PrintStack()
        t.Error(err)
      }
    },
    SSHConfig: &conf,
  }
  go srv.ServeNoClose(l)
  con, err := l.Connect()
  if err != nil {
    t.Fatal(err)
  }
  cl,err := Dial(con,&ssh.ClientConfig{
    User: "user",
    Auth: []ssh.AuthMethod{
      ssh.PublicKeys(cKey),
    },
  })
  if err != nil {
    t.Fatal(err)
  }
  rt := &ClientRoundTripper{Client: cl}
  req,err := http.NewRequest("GET","/foo",nil)
  res,err := rt.RoundTrip(req)
  if err != nil {
    t.Fatal(err)
  }
  if res.StatusCode != 204 {
    t.Fatalf("Wrong statuscode: %d",res.StatusCode)
  }
  cl.Close()
  l.Close()
  srv.Shutdown()
}
