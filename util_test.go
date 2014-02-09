package sshttp

import (
  "encoding/base64"
  "os"
  "io"
  "io/ioutil"
  "testing"
)

func TestAuthorizedKeyFile(t *testing.T){
  f,err  := ioutil.TempFile("","sshttp")
  if err != nil {
    t.Fatal(err)
  }
  fn := f.Name()
  defer os.Remove(fn)
  defer f.Close()
  io.WriteString(f,cKey.PublicKey().Type())
  io.WriteString(f," ")
  io.WriteString(f,base64.StdEncoding.EncodeToString(cKey.PublicKey().Marshal()))
  io.WriteString(f,"\n")
  f.Close()
  af := AuthorizedKeyFile{ Path: fn }
  err = af.Load()
  if err != nil {
    t.Fatal(err)
  }
  if len(af.Keys) != 1 {
    t.Fatalf("Wrong number of keys read: %d instead of 1",len(af.Keys))
  }
}

func TestSignerFromFile(t *testing.T){
  f,err  := ioutil.TempFile("","sshttp")
  if err != nil {
    t.Fatal(err)
  }
  fn := f.Name()
  defer os.Remove(fn)
  defer f.Close()
  io.WriteString(f,cKey.PublicKey().Type())
  io.WriteString(f," ")
  io.WriteString(f,base64.StdEncoding.EncodeToString(cKey.PublicKey().Marshal()))
  io.WriteString(f,"\n")
  f.Close()
  af := AuthorizedKeyFile{ Path: fn }
  err = af.Load()
  if err != nil {
    t.Fatal(err)
  }
  if len(af.Keys) != 1 {
    t.Fatalf("Wrong number of keys read: %d instead of 1",len(af.Keys))
  }
}

