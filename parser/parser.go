package parser

type Parser struct {
	state State
}

func NewParser() *Parser {
	p := new(Parser)
	p.state = ground{}
	return p
}

func appendTo(actions []Action, act Action) []Action {
	if !act.Ignore() {
		actions = append(actions, act)
	}
	return actions
}

// it's uesed to be input
func (p *Parser) parse(actions []Action, r rune) []Action {

	// start to parse
	ts := p.state.parse(r)

	// exit action from old state
	if ts.nextState != nil {
		actions = appendTo(actions, p.state.exit())
	}

	// transition action
	actions = appendTo(actions, ts.action)
	ts.action = nil

	// enter action to new state
	if ts.nextState != nil {
		actions = appendTo(actions, ts.nextState.enter())
		// transition to next state
		p.state = ts.nextState
	}

	return actions
}
