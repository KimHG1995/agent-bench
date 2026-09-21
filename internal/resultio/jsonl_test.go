package resultio

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/KimHG1995/agent-bench/internal/domain"
)

func TestWriterRefusesOverwriteAndPersistsEachRun(t *testing.T){
	path:=filepath.Join(t.TempDir(),"r.jsonl")
	w,err:=NewWriter(path,false);if err!=nil{t.Fatal(err)}
	if err:=w.Write(domain.RunResult{TaskID:"a"});err!=nil{t.Fatal(err)}
	if err:=w.Close();err!=nil{t.Fatal(err)}
	if _,err:=NewWriter(path,false);err==nil{t.Fatal("expected existing file error")}
	rows,err:=ReadJSONL(path);if err!=nil{t.Fatal(err)}
	if len(rows)!=1||rows[0].TaskID!="a"{t.Fatalf("rows=%#v",rows)}
	if info,err:=os.Stat(path);err!=nil||info.Size()==0{t.Fatal("result was not persisted")}
}
