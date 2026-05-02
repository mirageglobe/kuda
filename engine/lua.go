package engine

import (
	"fmt"
	"os"
	"sync"

	lua "github.com/yuin/gopher-lua"
)

// ScriptEngine wraps the Lua VM and provides an API for triggers and aliases.
// mu serialises all LState access; LState is not goroutine-safe.
type ScriptEngine struct {
	mu     sync.Mutex
	l      *lua.LState
	engine *Engine
}

func newScriptEngine(e *Engine) *ScriptEngine {
	l := lua.NewState()
	se := &ScriptEngine{l: l, engine: e}

	l.SetGlobal("send", l.NewFunction(se.luaSend))

	if _, err := os.Stat("scripts/init.lua"); err == nil {
		if err := l.DoFile("scripts/init.lua"); err != nil {
			fmt.Fprintf(os.Stderr, "Lua error: %v\n", err)
		}
	}

	return se
}

func (se *ScriptEngine) Close() {
	se.mu.Lock()
	defer se.mu.Unlock()
	se.l.Close()
}

func (se *ScriptEngine) luaSend(L *lua.LState) int {
	cmd := L.CheckString(1)
	if err := se.engine.source.Write([]byte(cmd + "\n")); err != nil {
		L.Error(lua.LString(err.Error()), 1)
	}
	return 0
}

func (se *ScriptEngine) evalAlias(cmd string) (string, bool) {
	se.mu.Lock()
	defer se.mu.Unlock()
	l := se.l
	fn := l.GetGlobal("onAlias")
	if fn.Type() != lua.LTFunction {
		return cmd, false
	}

	err := l.CallByParam(lua.P{
		Fn:      fn,
		NRet:    2,
		Protect: true,
	}, lua.LString(cmd))

	if err != nil {
		fmt.Fprintf(os.Stderr, "Lua alias error: %v\n", err)
		return cmd, false
	}

	ret2 := l.Get(-1)
	ret1 := l.Get(-2)
	l.Pop(2)

	consumed := false
	if b, ok := ret2.(lua.LBool); ok {
		consumed = bool(b)
	}
	newCmd := cmd
	if s, ok := ret1.(lua.LString); ok {
		newCmd = string(s)
	}

	return newCmd, consumed
}

func (se *ScriptEngine) evalTrigger(line string) {
	se.mu.Lock()
	defer se.mu.Unlock()
	l := se.l
	fn := l.GetGlobal("onText")
	if fn.Type() != lua.LTFunction {
		return
	}

	err := l.CallByParam(lua.P{
		Fn:      fn,
		NRet:    0,
		Protect: true,
	}, lua.LString(line))

	if err != nil {
		fmt.Fprintf(os.Stderr, "Lua trigger error: %v\n", err)
	}
}
