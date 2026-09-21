package mcpstdio

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestClientLifecycleAndToolCall(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "server.py")
	content := `import json, sys
for line in sys.stdin:
    req=json.loads(line)
    method=req.get("method")
    if "id" not in req:
        continue
    if method=="initialize":
        result={"protocolVersion":"2025-11-25","capabilities":{"tools":{}},"serverInfo":{"name":"fake","version":"1"}}
    elif method=="tools/list":
        result={"tools":[{"name":"inspect_typescript_graph","description":"fake","inputSchema":{"type":"object"}}]}
    elif method=="tools/call":
        result={"structuredContent":{"result":{"type":"lookup","hits":[{"name":"A"}]}}}
    else:
        result={}
    print(json.dumps({"jsonrpc":"2.0","id":req["id"],"result":result}), flush=True)
`
	if err := os.WriteFile(script, []byte(content), 0o644); err != nil { t.Fatal(err) }

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	client, err := Start(ctx, ProcessConfig{Command:"python3",Args:[]string{script},Env:InheritEnv()})
	if err != nil { t.Fatal(err) }
	defer client.Close()

	tools, err := client.ListTools()
	if err != nil { t.Fatal(err) }
	if len(tools) != 1 || tools[0].Name != "inspect_typescript_graph" { t.Fatalf("tools=%#v", tools) }

	result, err := client.CallTool("inspect_typescript_graph", map[string]any{"x":"y"})
	if err != nil { t.Fatal(err) }
	if result.StructuredContent == nil { t.Fatal("missing structured content") }
}
