package config

import (
	"fmt"
	"math"
	"sort"
	"strconv"
)

const endSymbol rune = 1114112

/* The rule types inferred from the grammar are below. */
type pegRule uint8

const (
	ruleUnknown pegRule = iota
	rulestart
	rulestatements
	ruleline
	ruleSection
	ruleSectionName
	ruleStatement
	ruleName
	ruleString
	ruleSingleQuotedString
	ruleDoubleQuotedString
	ruleRawString
	ruleComment
	ruleEOF
	ruleEOL
	rules
	ruleAction0
	rulePegText
	ruleAction1
	ruleAction2
	ruleAction3
	ruleAction4
	ruleAction5
	ruleAction6

	rulePre
	ruleIn
	ruleSuf
)

var rul3s = [...]string{
	"Unknown",
	"start",
	"statements",
	"line",
	"Section",
	"SectionName",
	"Statement",
	"Name",
	"String",
	"SingleQuotedString",
	"DoubleQuotedString",
	"RawString",
	"Comment",
	"EOF",
	"EOL",
	"s",
	"Action0",
	"PegText",
	"Action1",
	"Action2",
	"Action3",
	"Action4",
	"Action5",
	"Action6",

	"Pre_",
	"_In_",
	"_Suf",
}

type tokenTree interface {
	Print()
	PrintSyntax()
	PrintSyntaxTree(buffer string)
	Add(rule pegRule, begin, end, next uint32, depth int)
	Expand(index int) tokenTree
	Tokens() <-chan token32
	AST() *node32
	Error() []token32
	trim(length int)
}

type node32 struct {
	token32
	up, next *node32
}

func (node *node32) print(depth int, buffer string) {
	for node != nil {
		for c := 0; c < depth; c++ {
			fmt.Printf(" ")
		}
		fmt.Printf("\x1B[34m%v\x1B[m %v\n", rul3s[node.pegRule], strconv.Quote(string(([]rune(buffer)[node.begin:node.end]))))
		if node.up != nil {
			node.up.print(depth+1, buffer)
		}
		node = node.next
	}
}

func (node *node32) Print(buffer string) {
	node.print(0, buffer)
}

type element struct {
	node *node32
	down *element
}

/* ${@} bit structure for abstract syntax tree */
type token32 struct {
	pegRule
	begin, end, next uint32
}

func (t *token32) isZero() bool {
	return t.pegRule == ruleUnknown && t.begin == 0 && t.end == 0 && t.next == 0
}

func (t *token32) isParentOf(u token32) bool {
	return t.begin <= u.begin && t.end >= u.end && t.next > u.next
}

func (t *token32) getToken32() token32 {
	return token32{pegRule: t.pegRule, begin: uint32(t.begin), end: uint32(t.end), next: uint32(t.next)}
}

func (t *token32) String() string {
	return fmt.Sprintf("\x1B[34m%v\x1B[m %v %v %v", rul3s[t.pegRule], t.begin, t.end, t.next)
}

type tokens32 struct {
	tree    []token32
	ordered [][]token32
}

func (t *tokens32) trim(length int) {
	t.tree = t.tree[0:length]
}

func (t *tokens32) Print() {
	for _, token := range t.tree {
		fmt.Println(token.String())
	}
}

func (t *tokens32) Order() [][]token32 {
	if t.ordered != nil {
		return t.ordered
	}

	depths := make([]int32, 1, math.MaxInt16)
	for i, token := range t.tree {
		if token.pegRule == ruleUnknown {
			t.tree = t.tree[:i]
			break
		}
		depth := int(token.next)
		if length := len(depths); depth >= length {
			depths = depths[:depth+1]
		}
		depths[depth]++
	}
	depths = append(depths, 0)

	ordered, pool := make([][]token32, len(depths)), make([]token32, len(t.tree)+len(depths))
	for i, depth := range depths {
		depth++
		ordered[i], pool, depths[i] = pool[:depth], pool[depth:], 0
	}

	for i, token := range t.tree {
		depth := token.next
		token.next = uint32(i)
		ordered[depth][depths[depth]] = token
		depths[depth]++
	}
	t.ordered = ordered
	return ordered
}

type state32 struct {
	token32
	depths []int32
	leaf   bool
}

func (t *tokens32) AST() *node32 {
	tokens := t.Tokens()
	stack := &element{node: &node32{token32: <-tokens}}
	for token := range tokens {
		if token.begin == token.end {
			continue
		}
		node := &node32{token32: token}
		for stack != nil && stack.node.begin >= token.begin && stack.node.end <= token.end {
			stack.node.next = node.up
			node.up = stack.node
			stack = stack.down
		}
		stack = &element{node: node, down: stack}
	}
	return stack.node
}

func (t *tokens32) PreOrder() (<-chan state32, [][]token32) {
	s, ordered := make(chan state32, 6), t.Order()
	go func() {
		var states [8]state32
		for i := range states {
			states[i].depths = make([]int32, len(ordered))
		}
		depths, state, depth := make([]int32, len(ordered)), 0, 1
		write := func(t token32, leaf bool) {
			S := states[state]
			state, S.pegRule, S.begin, S.end, S.next, S.leaf = (state+1)%8, t.pegRule, t.begin, t.end, uint32(depth), leaf
			copy(S.depths, depths)
			s <- S
		}

		states[state].token32 = ordered[0][0]
		depths[0]++
		state++
		a, b := ordered[depth-1][depths[depth-1]-1], ordered[depth][depths[depth]]
	depthFirstSearch:
		for {
			for {
				if i := depths[depth]; i > 0 {
					if c, j := ordered[depth][i-1], depths[depth-1]; a.isParentOf(c) &&
						(j < 2 || !ordered[depth-1][j-2].isParentOf(c)) {
						if c.end != b.begin {
							write(token32{pegRule: ruleIn, begin: c.end, end: b.begin}, true)
						}
						break
					}
				}

				if a.begin < b.begin {
					write(token32{pegRule: rulePre, begin: a.begin, end: b.begin}, true)
				}
				break
			}

			next := depth + 1
			if c := ordered[next][depths[next]]; c.pegRule != ruleUnknown && b.isParentOf(c) {
				write(b, false)
				depths[depth]++
				depth, a, b = next, b, c
				continue
			}

			write(b, true)
			depths[depth]++
			c, parent := ordered[depth][depths[depth]], true
			for {
				if c.pegRule != ruleUnknown && a.isParentOf(c) {
					b = c
					continue depthFirstSearch
				} else if parent && b.end != a.end {
					write(token32{pegRule: ruleSuf, begin: b.end, end: a.end}, true)
				}

				depth--
				if depth > 0 {
					a, b, c = ordered[depth-1][depths[depth-1]-1], a, ordered[depth][depths[depth]]
					parent = a.isParentOf(b)
					continue
				}

				break depthFirstSearch
			}
		}

		close(s)
	}()
	return s, ordered
}

func (t *tokens32) PrintSyntax() {
	tokens, ordered := t.PreOrder()
	max := -1
	for token := range tokens {
		if !token.leaf {
			fmt.Printf("%v", token.begin)
			for i, leaf, depths := 0, int(token.next), token.depths; i < leaf; i++ {
				fmt.Printf(" \x1B[36m%v\x1B[m", rul3s[ordered[i][depths[i]-1].pegRule])
			}
			fmt.Printf(" \x1B[36m%v\x1B[m\n", rul3s[token.pegRule])
		} else if token.begin == token.end {
			fmt.Printf("%v", token.begin)
			for i, leaf, depths := 0, int(token.next), token.depths; i < leaf; i++ {
				fmt.Printf(" \x1B[31m%v\x1B[m", rul3s[ordered[i][depths[i]-1].pegRule])
			}
			fmt.Printf(" \x1B[31m%v\x1B[m\n", rul3s[token.pegRule])
		} else {
			for c, end := token.begin, token.end; c < end; c++ {
				if i := int(c); max+1 < i {
					for j := max; j < i; j++ {
						fmt.Printf("skip %v %v\n", j, token.String())
					}
					max = i
				} else if i := int(c); i <= max {
					for j := i; j <= max; j++ {
						fmt.Printf("dupe %v %v\n", j, token.String())
					}
				} else {
					max = int(c)
				}
				fmt.Printf("%v", c)
				for i, leaf, depths := 0, int(token.next), token.depths; i < leaf; i++ {
					fmt.Printf(" \x1B[34m%v\x1B[m", rul3s[ordered[i][depths[i]-1].pegRule])
				}
				fmt.Printf(" \x1B[34m%v\x1B[m\n", rul3s[token.pegRule])
			}
			fmt.Printf("\n")
		}
	}
}

func (t *tokens32) PrintSyntaxTree(buffer string) {
	tokens, _ := t.PreOrder()
	for token := range tokens {
		for c := 0; c < int(token.next); c++ {
			fmt.Printf(" ")
		}
		fmt.Printf("\x1B[34m%v\x1B[m %v\n", rul3s[token.pegRule], strconv.Quote(string(([]rune(buffer)[token.begin:token.end]))))
	}
}

func (t *tokens32) Add(rule pegRule, begin, end, depth uint32, index int) {
	t.tree[index] = token32{pegRule: rule, begin: uint32(begin), end: uint32(end), next: uint32(depth)}
}

func (t *tokens32) Tokens() <-chan token32 {
	s := make(chan token32, 16)
	go func() {
		for _, v := range t.tree {
			s <- v.getToken32()
		}
		close(s)
	}()
	return s
}

func (t *tokens32) Error() []token32 {
	ordered := t.Order()
	length := len(ordered)
	tokens, length := make([]token32, length), length-1
	for i := range tokens {
		o := ordered[length-i]
		if len(o) > 1 {
			tokens[i] = o[len(o)-2].getToken32()
		}
	}
	return tokens
}

/*func (t *tokens16) Expand(index int) tokenTree {
	tree := t.tree
	if index >= len(tree) {
		expanded := make([]token32, 2 * len(tree))
		for i, v := range tree {
			expanded[i] = v.getToken32()
		}
		return &tokens32{tree: expanded}
	}
	return nil
}*/

func (t *tokens32) Expand(index int) tokenTree {
	tree := t.tree
	if index >= len(tree) {
		expanded := make([]token32, 2*len(tree))
		copy(expanded, tree)
		t.tree = expanded
	}
	return nil
}

type erpelParser struct {
	configState

	Buffer string
	buffer []rune
	rules  [24]func() bool
	Parse  func(rule ...int) error
	Reset  func()
	Pretty bool
	tokenTree
}

type textPosition struct {
	line, symbol int
}

type textPositionMap map[int]textPosition

func translatePositions(buffer []rune, positions []int) textPositionMap {
	length, translations, j, line, symbol := len(positions), make(textPositionMap, len(positions)), 0, 1, 0
	sort.Ints(positions)

search:
	for i, c := range buffer {
		if c == '\n' {
			line, symbol = line+1, 0
		} else {
			symbol++
		}
		if i == positions[j] {
			translations[positions[j]] = textPosition{line, symbol}
			for j++; j < length; j++ {
				if i != positions[j] {
					continue search
				}
			}
			break search
		}
	}

	return translations
}

type parseError struct {
	p   *erpelParser
	max token32
}

func (e *parseError) Error() string {
	tokens, error := []token32{e.max}, "\n"
	positions, p := make([]int, 2*len(tokens)), 0
	for _, token := range tokens {
		positions[p], p = int(token.begin), p+1
		positions[p], p = int(token.end), p+1
	}
	translations := translatePositions(e.p.buffer, positions)
	format := "parse error near %v (line %v symbol %v - line %v symbol %v):\n%v\n"
	if e.p.Pretty {
		format = "parse error near \x1B[34m%v\x1B[m (line %v symbol %v - line %v symbol %v):\n%v\n"
	}
	for _, token := range tokens {
		begin, end := int(token.begin), int(token.end)
		error += fmt.Sprintf(format,
			rul3s[token.pegRule],
			translations[begin].line, translations[begin].symbol,
			translations[end].line, translations[end].symbol,
			strconv.Quote(string(e.p.buffer[begin:end])))
	}

	return error
}

func (p *erpelParser) PrintSyntaxTree() {
	p.tokenTree.PrintSyntaxTree(p.Buffer)
}

func (p *erpelParser) Highlighter() {
	p.tokenTree.PrintSyntax()
}

func (p *erpelParser) Execute() {
	buffer, _buffer, text, begin, end := p.Buffer, p.buffer, "", 0, 0
	for token := range p.tokenTree.Tokens() {
		switch token.pegRule {

		case rulePegText:
			begin, end = int(token.begin), int(token.end)
			text = string(_buffer[begin:end])

		case ruleAction0:
			p.setDefaultSection()
		case ruleAction1:
			p.newSection(buffer[begin:end])
		case ruleAction2:
			p.set(p.name, p.value)
		case ruleAction3:
			p.name = buffer[begin:end]
		case ruleAction4:
			p.value = buffer[begin:end]
		case ruleAction5:
			p.value = buffer[begin:end]
		case ruleAction6:
			p.value = buffer[begin:end]

		}
	}
	_, _, _, _, _ = buffer, _buffer, text, begin, end
}

func (p *erpelParser) Init() {
	p.buffer = []rune(p.Buffer)
	if len(p.buffer) == 0 || p.buffer[len(p.buffer)-1] != endSymbol {
		p.buffer = append(p.buffer, endSymbol)
	}

	var tree tokenTree = &tokens32{tree: make([]token32, math.MaxInt16)}
	var max token32
	position, depth, tokenIndex, buffer, _rules := uint32(0), uint32(0), 0, p.buffer, p.rules

	p.Parse = func(rule ...int) error {
		r := 1
		if len(rule) > 0 {
			r = rule[0]
		}
		matches := p.rules[r]()
		p.tokenTree = tree
		if matches {
			p.tokenTree.trim(tokenIndex)
			return nil
		}
		return &parseError{p, max}
	}

	p.Reset = func() {
		position, tokenIndex, depth = 0, 0, 0
	}

	add := func(rule pegRule, begin uint32) {
		if t := tree.Expand(tokenIndex); t != nil {
			tree = t
		}
		tree.Add(rule, begin, position, depth, tokenIndex)
		tokenIndex++
		if begin != position && position > max.end {
			max = token32{rule, begin, position, depth}
		}
	}

	matchDot := func() bool {
		if buffer[position] != endSymbol {
			position++
			return true
		}
		return false
	}

	/*matchChar := func(c byte) bool {
		if buffer[position] == c {
			position++
			return true
		}
		return false
	}*/

	/*matchRange := func(lower byte, upper byte) bool {
		if c := buffer[position]; c >= lower && c <= upper {
			position++
			return true
		}
		return false
	}*/

	_rules = [...]func() bool{
		nil,
		/* 0 start <- <(s statements s Section* statements EOF)> */
		func() bool {
			position0, tokenIndex0, depth0 := position, tokenIndex, depth
			{
				position1 := position
				depth++
				if !_rules[rules]() {
					goto l0
				}
				if !_rules[rulestatements]() {
					goto l0
				}
				if !_rules[rules]() {
					goto l0
				}
			l2:
				{
					position3, tokenIndex3, depth3 := position, tokenIndex, depth
					if !_rules[ruleSection]() {
						goto l3
					}
					goto l2
				l3:
					position, tokenIndex, depth = position3, tokenIndex3, depth3
				}
				if !_rules[rulestatements]() {
					goto l0
				}
				if !_rules[ruleEOF]() {
					goto l0
				}
				depth--
				add(rulestart, position1)
			}
			return true
		l0:
			position, tokenIndex, depth = position0, tokenIndex0, depth0
			return false
		},
		/* 1 statements <- <((line EOL)* line?)> */
		func() bool {
			{
				position5 := position
				depth++
			l6:
				{
					position7, tokenIndex7, depth7 := position, tokenIndex, depth
					if !_rules[ruleline]() {
						goto l7
					}
					if !_rules[ruleEOL]() {
						goto l7
					}
					goto l6
				l7:
					position, tokenIndex, depth = position7, tokenIndex7, depth7
				}
				{
					position8, tokenIndex8, depth8 := position, tokenIndex, depth
					if !_rules[ruleline]() {
						goto l8
					}
					goto l9
				l8:
					position, tokenIndex, depth = position8, tokenIndex8, depth8
				}
			l9:
				depth--
				add(rulestatements, position5)
			}
			return true
		},
		/* 2 line <- <(s (Comment / Statement)? s)> */
		func() bool {
			position10, tokenIndex10, depth10 := position, tokenIndex, depth
			{
				position11 := position
				depth++
				if !_rules[rules]() {
					goto l10
				}
				{
					position12, tokenIndex12, depth12 := position, tokenIndex, depth
					{
						position14, tokenIndex14, depth14 := position, tokenIndex, depth
						if !_rules[ruleComment]() {
							goto l15
						}
						goto l14
					l15:
						position, tokenIndex, depth = position14, tokenIndex14, depth14
						if !_rules[ruleStatement]() {
							goto l12
						}
					}
				l14:
					goto l13
				l12:
					position, tokenIndex, depth = position12, tokenIndex12, depth12
				}
			l13:
				if !_rules[rules]() {
					goto l10
				}
				depth--
				add(ruleline, position11)
			}
			return true
		l10:
			position, tokenIndex, depth = position10, tokenIndex10, depth10
			return false
		},
		/* 3 Section <- <(SectionName s '{' statements '}' s Action0)> */
		func() bool {
			position16, tokenIndex16, depth16 := position, tokenIndex, depth
			{
				position17 := position
				depth++
				if !_rules[ruleSectionName]() {
					goto l16
				}
				if !_rules[rules]() {
					goto l16
				}
				if buffer[position] != rune('{') {
					goto l16
				}
				position++
				if !_rules[rulestatements]() {
					goto l16
				}
				if buffer[position] != rune('}') {
					goto l16
				}
				position++
				if !_rules[rules]() {
					goto l16
				}
				if !_rules[ruleAction0]() {
					goto l16
				}
				depth--
				add(ruleSection, position17)
			}
			return true
		l16:
			position, tokenIndex, depth = position16, tokenIndex16, depth16
			return false
		},
		/* 4 SectionName <- <(<Name> Action1)> */
		func() bool {
			position18, tokenIndex18, depth18 := position, tokenIndex, depth
			{
				position19 := position
				depth++
				{
					position20 := position
					depth++
					if !_rules[ruleName]() {
						goto l18
					}
					depth--
					add(rulePegText, position20)
				}
				if !_rules[ruleAction1]() {
					goto l18
				}
				depth--
				add(ruleSectionName, position19)
			}
			return true
		l18:
			position, tokenIndex, depth = position18, tokenIndex18, depth18
			return false
		},
		/* 5 Statement <- <(Name s '=' s String Comment? Action2)> */
		func() bool {
			position21, tokenIndex21, depth21 := position, tokenIndex, depth
			{
				position22 := position
				depth++
				if !_rules[ruleName]() {
					goto l21
				}
				if !_rules[rules]() {
					goto l21
				}
				if buffer[position] != rune('=') {
					goto l21
				}
				position++
				if !_rules[rules]() {
					goto l21
				}
				if !_rules[ruleString]() {
					goto l21
				}
				{
					position23, tokenIndex23, depth23 := position, tokenIndex, depth
					if !_rules[ruleComment]() {
						goto l23
					}
					goto l24
				l23:
					position, tokenIndex, depth = position23, tokenIndex23, depth23
				}
			l24:
				if !_rules[ruleAction2]() {
					goto l21
				}
				depth--
				add(ruleStatement, position22)
			}
			return true
		l21:
			position, tokenIndex, depth = position21, tokenIndex21, depth21
			return false
		},
		/* 6 Name <- <(<([a-z] / [A-Z] / [0-9] / '-' / '_')+> Action3)> */
		func() bool {
			position25, tokenIndex25, depth25 := position, tokenIndex, depth
			{
				position26 := position
				depth++
				{
					position27 := position
					depth++
					{
						position30, tokenIndex30, depth30 := position, tokenIndex, depth
						if c := buffer[position]; c < rune('a') || c > rune('z') {
							goto l31
						}
						position++
						goto l30
					l31:
						position, tokenIndex, depth = position30, tokenIndex30, depth30
						if c := buffer[position]; c < rune('A') || c > rune('Z') {
							goto l32
						}
						position++
						goto l30
					l32:
						position, tokenIndex, depth = position30, tokenIndex30, depth30
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l33
						}
						position++
						goto l30
					l33:
						position, tokenIndex, depth = position30, tokenIndex30, depth30
						if buffer[position] != rune('-') {
							goto l34
						}
						position++
						goto l30
					l34:
						position, tokenIndex, depth = position30, tokenIndex30, depth30
						if buffer[position] != rune('_') {
							goto l25
						}
						position++
					}
				l30:
				l28:
					{
						position29, tokenIndex29, depth29 := position, tokenIndex, depth
						{
							position35, tokenIndex35, depth35 := position, tokenIndex, depth
							if c := buffer[position]; c < rune('a') || c > rune('z') {
								goto l36
							}
							position++
							goto l35
						l36:
							position, tokenIndex, depth = position35, tokenIndex35, depth35
							if c := buffer[position]; c < rune('A') || c > rune('Z') {
								goto l37
							}
							position++
							goto l35
						l37:
							position, tokenIndex, depth = position35, tokenIndex35, depth35
							if c := buffer[position]; c < rune('0') || c > rune('9') {
								goto l38
							}
							position++
							goto l35
						l38:
							position, tokenIndex, depth = position35, tokenIndex35, depth35
							if buffer[position] != rune('-') {
								goto l39
							}
							position++
							goto l35
						l39:
							position, tokenIndex, depth = position35, tokenIndex35, depth35
							if buffer[position] != rune('_') {
								goto l29
							}
							position++
						}
					l35:
						goto l28
					l29:
						position, tokenIndex, depth = position29, tokenIndex29, depth29
					}
					depth--
					add(rulePegText, position27)
				}
				if !_rules[ruleAction3]() {
					goto l25
				}
				depth--
				add(ruleName, position26)
			}
			return true
		l25:
			position, tokenIndex, depth = position25, tokenIndex25, depth25
			return false
		},
		/* 7 String <- <(DoubleQuotedString / SingleQuotedString / RawString)> */
		func() bool {
			position40, tokenIndex40, depth40 := position, tokenIndex, depth
			{
				position41 := position
				depth++
				{
					position42, tokenIndex42, depth42 := position, tokenIndex, depth
					if !_rules[ruleDoubleQuotedString]() {
						goto l43
					}
					goto l42
				l43:
					position, tokenIndex, depth = position42, tokenIndex42, depth42
					if !_rules[ruleSingleQuotedString]() {
						goto l44
					}
					goto l42
				l44:
					position, tokenIndex, depth = position42, tokenIndex42, depth42
					if !_rules[ruleRawString]() {
						goto l40
					}
				}
			l42:
				depth--
				add(ruleString, position41)
			}
			return true
		l40:
			position, tokenIndex, depth = position40, tokenIndex40, depth40
			return false
		},
		/* 8 SingleQuotedString <- <(<('\'' (('\\' '\'') / (!EOL !'\'' .))* '\'')> Action4)> */
		func() bool {
			position45, tokenIndex45, depth45 := position, tokenIndex, depth
			{
				position46 := position
				depth++
				{
					position47 := position
					depth++
					if buffer[position] != rune('\'') {
						goto l45
					}
					position++
				l48:
					{
						position49, tokenIndex49, depth49 := position, tokenIndex, depth
						{
							position50, tokenIndex50, depth50 := position, tokenIndex, depth
							if buffer[position] != rune('\\') {
								goto l51
							}
							position++
							if buffer[position] != rune('\'') {
								goto l51
							}
							position++
							goto l50
						l51:
							position, tokenIndex, depth = position50, tokenIndex50, depth50
							{
								position52, tokenIndex52, depth52 := position, tokenIndex, depth
								if !_rules[ruleEOL]() {
									goto l52
								}
								goto l49
							l52:
								position, tokenIndex, depth = position52, tokenIndex52, depth52
							}
							{
								position53, tokenIndex53, depth53 := position, tokenIndex, depth
								if buffer[position] != rune('\'') {
									goto l53
								}
								position++
								goto l49
							l53:
								position, tokenIndex, depth = position53, tokenIndex53, depth53
							}
							if !matchDot() {
								goto l49
							}
						}
					l50:
						goto l48
					l49:
						position, tokenIndex, depth = position49, tokenIndex49, depth49
					}
					if buffer[position] != rune('\'') {
						goto l45
					}
					position++
					depth--
					add(rulePegText, position47)
				}
				if !_rules[ruleAction4]() {
					goto l45
				}
				depth--
				add(ruleSingleQuotedString, position46)
			}
			return true
		l45:
			position, tokenIndex, depth = position45, tokenIndex45, depth45
			return false
		},
		/* 9 DoubleQuotedString <- <(<('"' (('\\' '"') / (!EOL !'"' .))* '"')> Action5)> */
		func() bool {
			position54, tokenIndex54, depth54 := position, tokenIndex, depth
			{
				position55 := position
				depth++
				{
					position56 := position
					depth++
					if buffer[position] != rune('"') {
						goto l54
					}
					position++
				l57:
					{
						position58, tokenIndex58, depth58 := position, tokenIndex, depth
						{
							position59, tokenIndex59, depth59 := position, tokenIndex, depth
							if buffer[position] != rune('\\') {
								goto l60
							}
							position++
							if buffer[position] != rune('"') {
								goto l60
							}
							position++
							goto l59
						l60:
							position, tokenIndex, depth = position59, tokenIndex59, depth59
							{
								position61, tokenIndex61, depth61 := position, tokenIndex, depth
								if !_rules[ruleEOL]() {
									goto l61
								}
								goto l58
							l61:
								position, tokenIndex, depth = position61, tokenIndex61, depth61
							}
							{
								position62, tokenIndex62, depth62 := position, tokenIndex, depth
								if buffer[position] != rune('"') {
									goto l62
								}
								position++
								goto l58
							l62:
								position, tokenIndex, depth = position62, tokenIndex62, depth62
							}
							if !matchDot() {
								goto l58
							}
						}
					l59:
						goto l57
					l58:
						position, tokenIndex, depth = position58, tokenIndex58, depth58
					}
					if buffer[position] != rune('"') {
						goto l54
					}
					position++
					depth--
					add(rulePegText, position56)
				}
				if !_rules[ruleAction5]() {
					goto l54
				}
				depth--
				add(ruleDoubleQuotedString, position55)
			}
			return true
		l54:
			position, tokenIndex, depth = position54, tokenIndex54, depth54
			return false
		},
		/* 10 RawString <- <(<('`' (!'`' .)* '`')> Action6)> */
		func() bool {
			position63, tokenIndex63, depth63 := position, tokenIndex, depth
			{
				position64 := position
				depth++
				{
					position65 := position
					depth++
					if buffer[position] != rune('`') {
						goto l63
					}
					position++
				l66:
					{
						position67, tokenIndex67, depth67 := position, tokenIndex, depth
						{
							position68, tokenIndex68, depth68 := position, tokenIndex, depth
							if buffer[position] != rune('`') {
								goto l68
							}
							position++
							goto l67
						l68:
							position, tokenIndex, depth = position68, tokenIndex68, depth68
						}
						if !matchDot() {
							goto l67
						}
						goto l66
					l67:
						position, tokenIndex, depth = position67, tokenIndex67, depth67
					}
					if buffer[position] != rune('`') {
						goto l63
					}
					position++
					depth--
					add(rulePegText, position65)
				}
				if !_rules[ruleAction6]() {
					goto l63
				}
				depth--
				add(ruleRawString, position64)
			}
			return true
		l63:
			position, tokenIndex, depth = position63, tokenIndex63, depth63
			return false
		},
		/* 11 Comment <- <(s '#' (!EOL .)*)> */
		func() bool {
			position69, tokenIndex69, depth69 := position, tokenIndex, depth
			{
				position70 := position
				depth++
				if !_rules[rules]() {
					goto l69
				}
				if buffer[position] != rune('#') {
					goto l69
				}
				position++
			l71:
				{
					position72, tokenIndex72, depth72 := position, tokenIndex, depth
					{
						position73, tokenIndex73, depth73 := position, tokenIndex, depth
						if !_rules[ruleEOL]() {
							goto l73
						}
						goto l72
					l73:
						position, tokenIndex, depth = position73, tokenIndex73, depth73
					}
					if !matchDot() {
						goto l72
					}
					goto l71
				l72:
					position, tokenIndex, depth = position72, tokenIndex72, depth72
				}
				depth--
				add(ruleComment, position70)
			}
			return true
		l69:
			position, tokenIndex, depth = position69, tokenIndex69, depth69
			return false
		},
		/* 12 EOF <- <!.> */
		func() bool {
			position74, tokenIndex74, depth74 := position, tokenIndex, depth
			{
				position75 := position
				depth++
				{
					position76, tokenIndex76, depth76 := position, tokenIndex, depth
					if !matchDot() {
						goto l76
					}
					goto l74
				l76:
					position, tokenIndex, depth = position76, tokenIndex76, depth76
				}
				depth--
				add(ruleEOF, position75)
			}
			return true
		l74:
			position, tokenIndex, depth = position74, tokenIndex74, depth74
			return false
		},
		/* 13 EOL <- <('\r' / '\n')> */
		func() bool {
			position77, tokenIndex77, depth77 := position, tokenIndex, depth
			{
				position78 := position
				depth++
				{
					position79, tokenIndex79, depth79 := position, tokenIndex, depth
					if buffer[position] != rune('\r') {
						goto l80
					}
					position++
					goto l79
				l80:
					position, tokenIndex, depth = position79, tokenIndex79, depth79
					if buffer[position] != rune('\n') {
						goto l77
					}
					position++
				}
			l79:
				depth--
				add(ruleEOL, position78)
			}
			return true
		l77:
			position, tokenIndex, depth = position77, tokenIndex77, depth77
			return false
		},
		/* 14 s <- <(' ' / '\t')*> */
		func() bool {
			{
				position82 := position
				depth++
			l83:
				{
					position84, tokenIndex84, depth84 := position, tokenIndex, depth
					{
						position85, tokenIndex85, depth85 := position, tokenIndex, depth
						if buffer[position] != rune(' ') {
							goto l86
						}
						position++
						goto l85
					l86:
						position, tokenIndex, depth = position85, tokenIndex85, depth85
						if buffer[position] != rune('\t') {
							goto l84
						}
						position++
					}
				l85:
					goto l83
				l84:
					position, tokenIndex, depth = position84, tokenIndex84, depth84
				}
				depth--
				add(rules, position82)
			}
			return true
		},
		/* 16 Action0 <- <{ p.setDefaultSection() }> */
		func() bool {
			{
				add(ruleAction0, position)
			}
			return true
		},
		nil,
		/* 18 Action1 <- <{ p.newSection(buffer[begin:end]) }> */
		func() bool {
			{
				add(ruleAction1, position)
			}
			return true
		},
		/* 19 Action2 <- <{ p.set(p.name, p.value) }> */
		func() bool {
			{
				add(ruleAction2, position)
			}
			return true
		},
		/* 20 Action3 <- <{ p.name = buffer[begin:end] }> */
		func() bool {
			{
				add(ruleAction3, position)
			}
			return true
		},
		/* 21 Action4 <- <{ p.value = buffer[begin:end] }> */
		func() bool {
			{
				add(ruleAction4, position)
			}
			return true
		},
		/* 22 Action5 <- <{ p.value = buffer[begin:end] }> */
		func() bool {
			{
				add(ruleAction5, position)
			}
			return true
		},
		/* 23 Action6 <- <{ p.value = buffer[begin:end] }> */
		func() bool {
			{
				add(ruleAction6, position)
			}
			return true
		},
	}
	p.rules = _rules
}
