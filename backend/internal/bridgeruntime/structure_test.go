package bridgeruntime

import (
	"context"
	"errors"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/devicebridge"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRuntimeNeverRedeems(t *testing.T) {
	files, e := filepath.Glob("*.go")
	if e != nil {
		t.Fatal(e)
	}
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		tree, e := parser.ParseFile(token.NewFileSet(), file, nil, 0)
		if e != nil {
			t.Fatal(e)
		}
		ast.Inspect(tree, func(n ast.Node) bool {
			if id, ok := n.(*ast.SelectorExpr); ok {
				switch id.Sel.Name {
				case "PairingCoordinator", "NewPairingCoordinator", "Pair", "RedeemBody", "Reserve", "Complete", "Block":
					t.Errorf("forbidden runtime reference %s", id.Sel.Name)
				}
			}
			return true
		})
		if _, e = os.Stat(file); e != nil {
			t.Fatal(e)
		}
	}
	f, d := factoryHarness(t)
	var calls, redeems int
	ctx, cancel := context.WithCancel(context.Background())
	f.deps.Dialer = dialFunc(func(_ context.Context, target string, _ http.Header) (devicebridge.Socket, error) {
		calls++
		if strings.Contains(target, devicebridge.RedeemPath) {
			redeems++
		}
		return nil, errors.New("offline")
	})
	f.deps.Sleep = func(context.Context, time.Duration) error {
		if calls == 3 {
			cancel()
		}
		return nil
	}
	raw, e := f.New(ctx, "https://example.test", d, nil)
	if e != nil {
		t.Fatal(e)
	}
	raw.Run(ctx)
	if calls != 3 || redeems != 0 {
		t.Fatal("runtime redeemed")
	}
}
