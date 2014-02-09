package sshttp

import (
  "net"
  "bytes"
  "io"
  "os"
  "sync"
  "code.google.com/p/gosshnew/ssh"
)

func Dial(conn net.Conn,config *ssh.ClientConfig) (*ssh.Client,error) {
  c, chans, reqs, err := ssh.NewClientConn(conn, conn.RemoteAddr().String(), config)
  if err != nil {
    return nil, err
  }
  return ssh.NewClient(c, chans, reqs), nil
}

type PublicKeyCallback func(conn ssh.ConnMetadata, algo string, pubkey []byte) bool

func AcceptPublicKey(keys ...ssh.PublicKey) PublicKeyCallback {
  return func(_ ssh.ConnMetadata, algo string, pubkey []byte) bool {
    for _, key := range(keys) {
      if key.PublicKeyAlgo() == algo && bytes.Compare(pubkey, key.Marshal()) == 0 {
        return true
      }
    }
    return false
  }
}

type AuthorizedKeyFile struct {
  Path string
  Keys []ssh.PublicKey

  mux sync.Mutex
}

func (a *AuthorizedKeyFile) Accept(conn ssh.ConnMetadata, algo string, pubkey []byte) bool {
  return AcceptPublicKey(a.Keys...)(conn, algo, pubkey)
}

func (a *AuthorizedKeyFile) Load() error {
  a.mux.Lock()
  defer a.mux.Unlock()
  file, err := os.Open(a.Path)
  if err != nil {
    return err
  }
  defer file.Close()
  r := &io.LimitedReader{ file, 10 * 1024 * 1024 }
  var buf bytes.Buffer
  _, err = io.Copy(&buf, r)
  if err != nil {
    return err
  }
  rest := buf.Bytes()
  var keys []ssh.PublicKey
  for len(rest) > 0 {
    var key ssh.PublicKey
    key,_,_,rest,err = ssh.ParseAuthorizedKey(rest)
    if err != nil {
      if err.Error() == "ssh: no key found" {
        break
      }
      return err
    }
    keys = append(keys,key)
  }
  a.Keys = keys
  return nil
}

func SignerFromFile(path string) (ssh.Signer, err error){
  file, err := os.Open(path)
  if err != nil {
    return
  }
  defer file.Close()
  r := &io.LimitedReader{ file, 1024 * 1024 }
  var buf bytes.Buffer
  _, err = io.Copy(&buf, r)
  if err != nil {
    return
  }
  return ssh.ParsePrivateKey(buf)
}
