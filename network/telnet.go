package network

type parserState int

const (
	stateText parserState = iota
	stateIAC
	stateDO
	stateDONT
	stateWILL
	stateWONT
	stateSB
	stateSBIAC
)

// telnetParser holds per-connection telnet state machine state.
type telnetParser struct {
	client *Client
	state  parserState
	sbBuf  []byte
}

func (p *telnetParser) handleByte(b byte) {
	switch p.state {
	case stateText:
		if b == IAC {
			p.state = stateIAC
		} else {
			p.client.events <- Event{Type: EventText, Data: []byte{b}}
		}

	case stateIAC:
		switch b {
		case IAC:
			p.client.events <- Event{Type: EventText, Data: []byte{IAC}}
			p.state = stateText
		case GA:
			p.client.events <- Event{Type: EventTelnetCommand, Data: []byte{GA}}
			p.state = stateText
		case DO:
			p.state = stateDO
		case DONT:
			p.state = stateDONT
		case WILL:
			p.state = stateWILL
		case WONT:
			p.state = stateWONT
		case SB:
			p.state = stateSB
			p.sbBuf = nil
		default:
			p.client.events <- Event{Type: EventTelnetCommand, Data: []byte{b}}
			p.state = stateText
		}

	case stateDO:
		p.handleNegotiation(DO, b)
		p.state = stateText

	case stateDONT:
		p.handleNegotiation(DONT, b)
		p.state = stateText

	case stateWILL:
		p.handleNegotiation(WILL, b)
		p.state = stateText

	case stateWONT:
		p.handleNegotiation(WONT, b)
		p.state = stateText

	case stateSB:
		if b == IAC {
			p.state = stateSBIAC
		} else {
			p.sbBuf = append(p.sbBuf, b)
		}

	case stateSBIAC:
		if b == SE {
			p.handleSubnegotiation(p.sbBuf)
			p.state = stateText
		} else if b == IAC {
			p.sbBuf = append(p.sbBuf, IAC)
			p.state = stateSB
		} else {
			// malformed SB sequence — absorb and continue
			p.sbBuf = append(p.sbBuf, IAC, b)
			p.state = stateSB
		}
	}
}

func (p *telnetParser) handleNegotiation(cmd, option byte) {
	// handle supported options explicitly
	switch cmd {
	case WILL:
		if option == TelnetOptionEcho {
			p.client.Write([]byte{IAC, DO, option}) //nolint:errcheck
			p.client.events <- Event{Type: EventTelnetCommand, Data: []byte{cmd, option}}
			return
		}
		p.client.Write([]byte{IAC, DONT, option}) //nolint:errcheck
	case WONT:
		if option == TelnetOptionEcho {
			p.client.Write([]byte{IAC, DONT, option}) //nolint:errcheck
			p.client.events <- Event{Type: EventTelnetCommand, Data: []byte{cmd, option}}
			return
		}
	case DO:
		p.client.Write([]byte{IAC, WONT, option}) //nolint:errcheck
	}
	p.client.events <- Event{Type: EventTelnetCommand, Data: []byte{cmd, option}}
}

func (p *telnetParser) handleSubnegotiation(data []byte) {
	if len(data) > 0 && data[0] == TelnetOptionGMCP {
		p.client.events <- Event{Type: EventGMCP, Data: data[1:]}
	}
}
