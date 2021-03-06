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
    HostKeyCallback: AcceptHostKey(sKey.PublicKey()),
  }
}

func genServer(h ChannelHandler,t *testing.T) *Server{
  return &Server{
    Handler: h,
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
    HttpHandlerFunc(func(wr http.ResponseWriter,req *http.Request){
      wr.WriteHeader(200)
      io.WriteString(wr,"bar")
    }),t)
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
    if res.StatusCode != 200 {
      t.Fatalf("Wrong statuscode: %d",res.StatusCode)
    }
    b := make([]byte,3)
    if _,err := io.ReadFull(res.Body,b) ; err != nil {
      t.Fatal(err)
    }
    if string(b) != "bar" {
      t.Fatalf("Expect \"bar\", got %s",string(b))
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
    HttpHandlerFunc(func(wr http.ResponseWriter,req *http.Request){
      wr.WriteHeader(204)
    }),t)
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

