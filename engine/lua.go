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
	L      *lua.LState
	engine *Engine
}

func newScriptEngine(e *Engine) *ScriptEngine {
	L := lua.NewState()
	se := &ScriptEngine{L: L, engine: e}

	// Register Go functions to Lua
	L.SetGlobal("send", L.NewFunction(se.luaSend))

	// Load user script if it exists
	if _, err := os.Stat("scripts/init.lua"); err == nil {
		if err := L.DoFile("scripts/init.lua"); err != nil {
			fmt.Printf("Lua error: %v\n", err)
		}
	}

	return se
}

func (se *ScriptEngine) Close() {
	se.mu.Lock()
	defer se.mu.Unlock()
	se.L.Close()
}

// luaSend is exposed to Lua as send(cmd)
func (se *ScriptEngine) luaSend(L *lua.LState) int {
	cmd := L.CheckString(1)
	if err := se.engine.source.Write([]byte(cmd + "\n")); err != nil {
		L.Error(lua.LString(err.Error()), 1)
	}
	return 0
}

// evalAlias evaluates a command through Lua aliases.
// Returns (modifiedCommand, consumed).
func (se *ScriptEngine) evalAlias(cmd string) (string, bool) {
	se.mu.Lock()
	defer se.mu.Unlock()
	L := se.L
	fn := L.GetGlobal("onAlias")
	if fn.Type() != lua.LTFunction {
		return cmd, false
	}

	err := L.CallByParam(lua.P{
		Fn:      fn,
		NRet:    2,
		Protect: true,
	}, lua.LString(cmd))

	if err != nil {
		fmt.Printf("Lua alias error: %v\n", err)
		return cmd, false
	}

	ret2 := L.Get(-1) // bool: consumed
	ret1 := L.Get(-2) // string: newCmd
	L.Pop(2)

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

// evalTrigger evaluates a line of text through Lua triggers.
func (se *ScriptEngine) evalTrigger(line string) {
	se.mu.Lock()
	defer se.mu.Unlock()
	L := se.L
	fn := L.GetGlobal("onText")
	if fn.Type() != lua.LTFunction {
		return
	}

	err := L.CallByParam(lua.P{
		Fn:      fn,
		NRet:    0,
		Protect: true,
	}, lua.LString(line))

	if err != nil {
		fmt.Printf("Lua trigger error: %v\n", err)
	}
}
