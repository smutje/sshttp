package sshttp

import (
  "errors"
  "bufio"
  "sync"
  "net/http"
  "code.google.com/p/gosshnew/ssh"
)

type RoundTripper struct {
  Config *ssh.ClientConfig
}

func (r *RoundTripper) RoundTrip(req *http.Request) (res *http.Response,err error){
  if req.URL.Host == "" {
    return nil, errors.New("Missing hostname for request")
  }
  client,err := ssh.Dial("tcp",req.URL.Host,r.Config)
  if err != nil {
    return
  }
  rt         := ClientRoundTripper{
    Client: client,
  }
  defer rt.Close()
  return rt.RoundTrip(req)
}

type ClientRoundTripper struct{
  Client *ssh.Client
  PoolSize uint

  once sync.Once
  pool chan ssh.Channel
}

func (rt *ClientRoundTripper) Close() error{
  rt.CloseIdleConnections()
  close(rt.pool)
  return rt.Client.Close()
}

func (rt *ClientRoundTripper) CloseIdleConnections(){
  for{
    select{
    case ch := <-rt.pool :
      ch.Close()
    default:
      return
    }
  }
}

func (rt *ClientRoundTripper) init(){
  rt.once.Do(func(){
    rt.pool = make(chan ssh.Channel,rt.PoolSize)
  })
}

func (rt *ClientRoundTripper) checkout() (ssh.Channel,error) {
  rt.init()
  select{
  case ch := <-rt.pool :
    return ch, nil
  default:
    ch, in ,err := rt.Client.OpenChannel("http",nil)
    go ssh.DiscardRequests(in)
    return ch, err
  }
}

func (rt *ClientRoundTripper) checkin(ch ssh.Channel) {
  rt.init()
  select{
  case rt.pool <- ch:
    return
  default:
    ch.Close()
  }
}

func (r *ClientRoundTripper) RoundTrip(req *http.Request) (*http.Response, error){
  ch, err := r.checkout()
  if err != nil {
    return nil, err
  }
  defer r.checkin(ch)
  err = req.Write(ch)
  if err != nil {
    return nil, err
  }
  buf := bufio.NewReader(ch)
  return http.ReadResponse(buf,req)
}

