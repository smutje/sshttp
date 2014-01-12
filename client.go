package sshttp

import (
  "bufio"
  "sync"
  "net/http"
  "code.google.com/p/gosshnew/ssh"
)


type ClientRoundTripper struct{
  Client *ssh.Client

  once sync.Once
  pool chan ssh.Channel
}

func (rt *ClientRoundTripper) init(){
  rt.once.Do(func(){
    rt.pool = make(chan ssh.Channel,5)
  })
}

func (rt *ClientRoundTripper) checkout() (ssh.Channel,error) {
  rt.init()
  select{
  case ch := <-rt.pool :
    return ch, nil
  default:
    ch, in ,err := rt.Client.OpenChannel("http",nil)
    go ssh.DiscardIncoming(in)
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

