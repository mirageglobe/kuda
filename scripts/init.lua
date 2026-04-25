-- KUDA Lua Configuration

print("[ Lua ] script initialized")

-- Aliases
-- onAlias(cmd) -> returns (newCmd, swallowed)
function onAlias(cmd)
    if cmd == "hi" then
        return "say Hello from Kuda Lua!", false
    end
    
    if cmd == "test" then
        send("say Testing the send() function")
        return "", true -- swallowed
    end

    return cmd, false
end

-- Triggers
-- onText(line)
function onText(line)
    if line:find("You are hungry") then
        send("eat bread")
    end
end
