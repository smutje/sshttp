package sshttp

import (
  "testing"
  "net"
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
  sConf ssh.ServerConfig
  cConf ssh.ClientConfig
)

func init(){
  sKey = genKey()
  cKey = genKey()
  sConf = ssh.ServerConfig{
    PublicKeyCallback: AcceptPublicKey(cKey.PublicKey()),
  }
  sConf.AddHostKey(sKey)
  cConf = ssh.ClientConfig{
    User: "user",
    Auth: []ssh.AuthMethod{
      ssh.PublicKeys(cKey),
    },
  }
}

func genServer(f func(wr http.ResponseWriter,req *http.Request),t *testing.T) *Server{
  return &Server{
    Handler: http.HandlerFunc(f),
    ErrorHandler: func(err error){
      if err == io.EOF {
        t.Log("Ignoring EOF")
      }else{
        debug.PrintStack()
        t.Error(err)
      }
    },
    SSHConfig: &sConf,
  }
}

func TestSimpleHttp(t *testing.T){
  l := Listen()
  srv := genServer(
    func(wr http.ResponseWriter,req *http.Request){
      wr.WriteHeader(204)
    },t)
  go srv.ServeNoClose(l)
  con, err := l.Connect()
  if err != nil {
    t.Fatal(err)
  }
  cl,err := Dial(con,&cConf)
  if err != nil {
    t.Fatal(err)
  }
  rt := &ClientRoundTripper{Client: cl, PoolSize: 5}
  for i := 0 ; i < 10 ; i++ {
    req,err := http.NewRequest("GET","/foo",nil)
    res,err := rt.RoundTrip(req)
    if err != nil {
      t.Fatal(err)
    }
    if res.StatusCode != 204 {
      t.Fatalf("Wrong statuscode: %d",res.StatusCode)
    }
  }
  cl.Close()
  l.Close()
  srv.Shutdown()
}

func TestRealRoundtripper(t *testing.T){
  l,err := net.Listen("tcp","localhost:2280")
  if err != nil {
    t.Fatal(err)
  }
  srv := genServer(
    func(wr http.ResponseWriter,req *http.Request){
      wr.WriteHeader(204)
    },t)
  go srv.ServeNoClose(l)
  rt := &RoundTripper{Config: &cConf}
  for i := 0 ; i < 10 ; i++ {
    req,err := http.NewRequest("GET","sshttp://localhost:2280/foo",nil)
    res,err := rt.RoundTrip(req)
    if err != nil {
      t.Fatal(err)
    }
    if res.StatusCode != 204 {
      t.Fatalf("Wrong statuscode: %d",res.StatusCode)
    }
  }
  l.Close()
  srv.Shutdown()
}

