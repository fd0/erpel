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
	ruleLine
	ruleName
	ruleStatement
	ruleField
	ruleFieldName
	ruleFieldData
	ruleFieldStatement
	ruleValue
	ruleString
	ruleList
	ruleSingleQuotedString
	ruleDoubleQuotedString
	ruleRawString
	ruleComment
	ruleEOF
	ruleEOL
	rules
	rulePegText
	ruleAction0
	ruleAction1
	ruleAction2
	ruleAction3
	ruleAction4
	ruleAction5
	ruleAction6
	ruleAction7

	rulePre
	ruleIn
	ruleSuf
)

var rul3s = [...]string{
	"Unknown",
	"start",
	"Line",
	"Name",
	"Statement",
	"Field",
	"FieldName",
	"FieldData",
	"FieldStatement",
	"Value",
	"String",
	"List",
	"SingleQuotedString",
	"DoubleQuotedString",
	"RawString",
	"Comment",
	"EOF",
	"EOL",
	"s",
	"PegText",
	"Action0",
	"Action1",
	"Action2",
	"Action3",
	"Action4",
	"Action5",
	"Action6",
	"Action7",

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
	State

	Buffer string
	buffer []rune
	rules  [28]func() bool
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
			p.name = buffer[begin:end]
		case ruleAction1:
			p.set(p.name, p.value)
		case ruleAction2:
			p.inField = false
		case ruleAction3:
			p.inField = true
			p.newField(buffer[begin:end])
		case ruleAction4:
			p.value = buffer[begin:end]
		case ruleAction5:
			p.value = buffer[begin:end]
		case ruleAction6:
			p.value = buffer[begin:end]
		case ruleAction7:
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
		/* 0 start <- <((Line EOL)* Line? EOF)> */
		func() bool {
			position0, tokenIndex0, depth0 := position, tokenIndex, depth
			{
				position1 := position
				depth++
			l2:
				{
					position3, tokenIndex3, depth3 := position, tokenIndex, depth
					if !_rules[ruleLine]() {
						goto l3
					}
					if !_rules[ruleEOL]() {
						goto l3
					}
					goto l2
				l3:
					position, tokenIndex, depth = position3, tokenIndex3, depth3
				}
				{
					position4, tokenIndex4, depth4 := position, tokenIndex, depth
					if !_rules[ruleLine]() {
						goto l4
					}
					goto l5
				l4:
					position, tokenIndex, depth = position4, tokenIndex4, depth4
				}
			l5:
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
		/* 1 Line <- <((Field / Statement)? s Comment?)> */
		func() bool {
			position6, tokenIndex6, depth6 := position, tokenIndex, depth
			{
				position7 := position
				depth++
				{
					position8, tokenIndex8, depth8 := position, tokenIndex, depth
					{
						position10, tokenIndex10, depth10 := position, tokenIndex, depth
						if !_rules[ruleField]() {
							goto l11
						}
						goto l10
					l11:
						position, tokenIndex, depth = position10, tokenIndex10, depth10
						if !_rules[ruleStatement]() {
							goto l8
						}
					}
				l10:
					goto l9
				l8:
					position, tokenIndex, depth = position8, tokenIndex8, depth8
				}
			l9:
				if !_rules[rules]() {
					goto l6
				}
				{
					position12, tokenIndex12, depth12 := position, tokenIndex, depth
					if !_rules[ruleComment]() {
						goto l12
					}
					goto l13
				l12:
					position, tokenIndex, depth = position12, tokenIndex12, depth12
				}
			l13:
				depth--
				add(ruleLine, position7)
			}
			return true
		l6:
			position, tokenIndex, depth = position6, tokenIndex6, depth6
			return false
		},
		/* 2 Name <- <(<([a-z] / [A-Z] / [0-9] / '-' / '_')+> Action0)> */
		func() bool {
			position14, tokenIndex14, depth14 := position, tokenIndex, depth
			{
				position15 := position
				depth++
				{
					position16 := position
					depth++
					{
						position19, tokenIndex19, depth19 := position, tokenIndex, depth
						if c := buffer[position]; c < rune('a') || c > rune('z') {
							goto l20
						}
						position++
						goto l19
					l20:
						position, tokenIndex, depth = position19, tokenIndex19, depth19
						if c := buffer[position]; c < rune('A') || c > rune('Z') {
							goto l21
						}
						position++
						goto l19
					l21:
						position, tokenIndex, depth = position19, tokenIndex19, depth19
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l22
						}
						position++
						goto l19
					l22:
						position, tokenIndex, depth = position19, tokenIndex19, depth19
						if buffer[position] != rune('-') {
							goto l23
						}
						position++
						goto l19
					l23:
						position, tokenIndex, depth = position19, tokenIndex19, depth19
						if buffer[position] != rune('_') {
							goto l14
						}
						position++
					}
				l19:
				l17:
					{
						position18, tokenIndex18, depth18 := position, tokenIndex, depth
						{
							position24, tokenIndex24, depth24 := position, tokenIndex, depth
							if c := buffer[position]; c < rune('a') || c > rune('z') {
								goto l25
							}
							position++
							goto l24
						l25:
							position, tokenIndex, depth = position24, tokenIndex24, depth24
							if c := buffer[position]; c < rune('A') || c > rune('Z') {
								goto l26
							}
							position++
							goto l24
						l26:
							position, tokenIndex, depth = position24, tokenIndex24, depth24
							if c := buffer[position]; c < rune('0') || c > rune('9') {
								goto l27
							}
							position++
							goto l24
						l27:
							position, tokenIndex, depth = position24, tokenIndex24, depth24
							if buffer[position] != rune('-') {
								goto l28
							}
							position++
							goto l24
						l28:
							position, tokenIndex, depth = position24, tokenIndex24, depth24
							if buffer[position] != rune('_') {
								goto l18
							}
							position++
						}
					l24:
						goto l17
					l18:
						position, tokenIndex, depth = position18, tokenIndex18, depth18
					}
					depth--
					add(rulePegText, position16)
				}
				if !_rules[ruleAction0]() {
					goto l14
				}
				depth--
				add(ruleName, position15)
			}
			return true
		l14:
			position, tokenIndex, depth = position14, tokenIndex14, depth14
			return false
		},
		/* 3 Statement <- <(s Name s '=' s Value Action1)> */
		func() bool {
			position29, tokenIndex29, depth29 := position, tokenIndex, depth
			{
				position30 := position
				depth++
				if !_rules[rules]() {
					goto l29
				}
				if !_rules[ruleName]() {
					goto l29
				}
				if !_rules[rules]() {
					goto l29
				}
				if buffer[position] != rune('=') {
					goto l29
				}
				position++
				if !_rules[rules]() {
					goto l29
				}
				if !_rules[ruleValue]() {
					goto l29
				}
				if !_rules[ruleAction1]() {
					goto l29
				}
				depth--
				add(ruleStatement, position30)
			}
			return true
		l29:
			position, tokenIndex, depth = position29, tokenIndex29, depth29
			return false
		},
		/* 4 Field <- <(s (('f' / 'F') ('i' / 'I') ('e' / 'E') ('l' / 'L') ('d' / 'D')) s FieldName s '{' FieldData '}' Action2)> */
		func() bool {
			position31, tokenIndex31, depth31 := position, tokenIndex, depth
			{
				position32 := position
				depth++
				if !_rules[rules]() {
					goto l31
				}
				{
					position33, tokenIndex33, depth33 := position, tokenIndex, depth
					if buffer[position] != rune('f') {
						goto l34
					}
					position++
					goto l33
				l34:
					position, tokenIndex, depth = position33, tokenIndex33, depth33
					if buffer[position] != rune('F') {
						goto l31
					}
					position++
				}
			l33:
				{
					position35, tokenIndex35, depth35 := position, tokenIndex, depth
					if buffer[position] != rune('i') {
						goto l36
					}
					position++
					goto l35
				l36:
					position, tokenIndex, depth = position35, tokenIndex35, depth35
					if buffer[position] != rune('I') {
						goto l31
					}
					position++
				}
			l35:
				{
					position37, tokenIndex37, depth37 := position, tokenIndex, depth
					if buffer[position] != rune('e') {
						goto l38
					}
					position++
					goto l37
				l38:
					position, tokenIndex, depth = position37, tokenIndex37, depth37
					if buffer[position] != rune('E') {
						goto l31
					}
					position++
				}
			l37:
				{
					position39, tokenIndex39, depth39 := position, tokenIndex, depth
					if buffer[position] != rune('l') {
						goto l40
					}
					position++
					goto l39
				l40:
					position, tokenIndex, depth = position39, tokenIndex39, depth39
					if buffer[position] != rune('L') {
						goto l31
					}
					position++
				}
			l39:
				{
					position41, tokenIndex41, depth41 := position, tokenIndex, depth
					if buffer[position] != rune('d') {
						goto l42
					}
					position++
					goto l41
				l42:
					position, tokenIndex, depth = position41, tokenIndex41, depth41
					if buffer[position] != rune('D') {
						goto l31
					}
					position++
				}
			l41:
				if !_rules[rules]() {
					goto l31
				}
				if !_rules[ruleFieldName]() {
					goto l31
				}
				if !_rules[rules]() {
					goto l31
				}
				if buffer[position] != rune('{') {
					goto l31
				}
				position++
				if !_rules[ruleFieldData]() {
					goto l31
				}
				if buffer[position] != rune('}') {
					goto l31
				}
				position++
				if !_rules[ruleAction2]() {
					goto l31
				}
				depth--
				add(ruleField, position32)
			}
			return true
		l31:
			position, tokenIndex, depth = position31, tokenIndex31, depth31
			return false
		},
		/* 5 FieldName <- <(<([a-z] / [A-Z] / [0-9] / '-' / '_')+> Action3)> */
		func() bool {
			position43, tokenIndex43, depth43 := position, tokenIndex, depth
			{
				position44 := position
				depth++
				{
					position45 := position
					depth++
					{
						position48, tokenIndex48, depth48 := position, tokenIndex, depth
						if c := buffer[position]; c < rune('a') || c > rune('z') {
							goto l49
						}
						position++
						goto l48
					l49:
						position, tokenIndex, depth = position48, tokenIndex48, depth48
						if c := buffer[position]; c < rune('A') || c > rune('Z') {
							goto l50
						}
						position++
						goto l48
					l50:
						position, tokenIndex, depth = position48, tokenIndex48, depth48
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l51
						}
						position++
						goto l48
					l51:
						position, tokenIndex, depth = position48, tokenIndex48, depth48
						if buffer[position] != rune('-') {
							goto l52
						}
						position++
						goto l48
					l52:
						position, tokenIndex, depth = position48, tokenIndex48, depth48
						if buffer[position] != rune('_') {
							goto l43
						}
						position++
					}
				l48:
				l46:
					{
						position47, tokenIndex47, depth47 := position, tokenIndex, depth
						{
							position53, tokenIndex53, depth53 := position, tokenIndex, depth
							if c := buffer[position]; c < rune('a') || c > rune('z') {
								goto l54
							}
							position++
							goto l53
						l54:
							position, tokenIndex, depth = position53, tokenIndex53, depth53
							if c := buffer[position]; c < rune('A') || c > rune('Z') {
								goto l55
							}
							position++
							goto l53
						l55:
							position, tokenIndex, depth = position53, tokenIndex53, depth53
							if c := buffer[position]; c < rune('0') || c > rune('9') {
								goto l56
							}
							position++
							goto l53
						l56:
							position, tokenIndex, depth = position53, tokenIndex53, depth53
							if buffer[position] != rune('-') {
								goto l57
							}
							position++
							goto l53
						l57:
							position, tokenIndex, depth = position53, tokenIndex53, depth53
							if buffer[position] != rune('_') {
								goto l47
							}
							position++
						}
					l53:
						goto l46
					l47:
						position, tokenIndex, depth = position47, tokenIndex47, depth47
					}
					depth--
					add(rulePegText, position45)
				}
				if !_rules[ruleAction3]() {
					goto l43
				}
				depth--
				add(ruleFieldName, position44)
			}
			return true
		l43:
			position, tokenIndex, depth = position43, tokenIndex43, depth43
			return false
		},
		/* 6 FieldData <- <((FieldStatement EOL)* FieldStatement?)> */
		func() bool {
			{
				position59 := position
				depth++
			l60:
				{
					position61, tokenIndex61, depth61 := position, tokenIndex, depth
					if !_rules[ruleFieldStatement]() {
						goto l61
					}
					if !_rules[ruleEOL]() {
						goto l61
					}
					goto l60
				l61:
					position, tokenIndex, depth = position61, tokenIndex61, depth61
				}
				{
					position62, tokenIndex62, depth62 := position, tokenIndex, depth
					if !_rules[ruleFieldStatement]() {
						goto l62
					}
					goto l63
				l62:
					position, tokenIndex, depth = position62, tokenIndex62, depth62
				}
			l63:
				depth--
				add(ruleFieldData, position59)
			}
			return true
		},
		/* 7 FieldStatement <- <(Statement? s Comment?)> */
		func() bool {
			position64, tokenIndex64, depth64 := position, tokenIndex, depth
			{
				position65 := position
				depth++
				{
					position66, tokenIndex66, depth66 := position, tokenIndex, depth
					if !_rules[ruleStatement]() {
						goto l66
					}
					goto l67
				l66:
					position, tokenIndex, depth = position66, tokenIndex66, depth66
				}
			l67:
				if !_rules[rules]() {
					goto l64
				}
				{
					position68, tokenIndex68, depth68 := position, tokenIndex, depth
					if !_rules[ruleComment]() {
						goto l68
					}
					goto l69
				l68:
					position, tokenIndex, depth = position68, tokenIndex68, depth68
				}
			l69:
				depth--
				add(ruleFieldStatement, position65)
			}
			return true
		l64:
			position, tokenIndex, depth = position64, tokenIndex64, depth64
			return false
		},
		/* 8 Value <- <(List / String)> */
		func() bool {
			position70, tokenIndex70, depth70 := position, tokenIndex, depth
			{
				position71 := position
				depth++
				{
					position72, tokenIndex72, depth72 := position, tokenIndex, depth
					if !_rules[ruleList]() {
						goto l73
					}
					goto l72
				l73:
					position, tokenIndex, depth = position72, tokenIndex72, depth72
					if !_rules[ruleString]() {
						goto l70
					}
				}
			l72:
				depth--
				add(ruleValue, position71)
			}
			return true
		l70:
			position, tokenIndex, depth = position70, tokenIndex70, depth70
			return false
		},
		/* 9 String <- <(DoubleQuotedString / SingleQuotedString / RawString)> */
		func() bool {
			position74, tokenIndex74, depth74 := position, tokenIndex, depth
			{
				position75 := position
				depth++
				{
					position76, tokenIndex76, depth76 := position, tokenIndex, depth
					if !_rules[ruleDoubleQuotedString]() {
						goto l77
					}
					goto l76
				l77:
					position, tokenIndex, depth = position76, tokenIndex76, depth76
					if !_rules[ruleSingleQuotedString]() {
						goto l78
					}
					goto l76
				l78:
					position, tokenIndex, depth = position76, tokenIndex76, depth76
					if !_rules[ruleRawString]() {
						goto l74
					}
				}
			l76:
				depth--
				add(ruleString, position75)
			}
			return true
		l74:
			position, tokenIndex, depth = position74, tokenIndex74, depth74
			return false
		},
		/* 10 List <- <(<('[' s (s String s ',' s)* s String s ']')> Action4)> */
		func() bool {
			position79, tokenIndex79, depth79 := position, tokenIndex, depth
			{
				position80 := position
				depth++
				{
					position81 := position
					depth++
					if buffer[position] != rune('[') {
						goto l79
					}
					position++
					if !_rules[rules]() {
						goto l79
					}
				l82:
					{
						position83, tokenIndex83, depth83 := position, tokenIndex, depth
						if !_rules[rules]() {
							goto l83
						}
						if !_rules[ruleString]() {
							goto l83
						}
						if !_rules[rules]() {
							goto l83
						}
						if buffer[position] != rune(',') {
							goto l83
						}
						position++
						if !_rules[rules]() {
							goto l83
						}
						goto l82
					l83:
						position, tokenIndex, depth = position83, tokenIndex83, depth83
					}
					if !_rules[rules]() {
						goto l79
					}
					if !_rules[ruleString]() {
						goto l79
					}
					if !_rules[rules]() {
						goto l79
					}
					if buffer[position] != rune(']') {
						goto l79
					}
					position++
					depth--
					add(rulePegText, position81)
				}
				if !_rules[ruleAction4]() {
					goto l79
				}
				depth--
				add(ruleList, position80)
			}
			return true
		l79:
			position, tokenIndex, depth = position79, tokenIndex79, depth79
			return false
		},
		/* 11 SingleQuotedString <- <(<('\'' (('\\' '\'') / (!EOL !'\'' .))* '\'')> Action5)> */
		func() bool {
			position84, tokenIndex84, depth84 := position, tokenIndex, depth
			{
				position85 := position
				depth++
				{
					position86 := position
					depth++
					if buffer[position] != rune('\'') {
						goto l84
					}
					position++
				l87:
					{
						position88, tokenIndex88, depth88 := position, tokenIndex, depth
						{
							position89, tokenIndex89, depth89 := position, tokenIndex, depth
							if buffer[position] != rune('\\') {
								goto l90
							}
							position++
							if buffer[position] != rune('\'') {
								goto l90
							}
							position++
							goto l89
						l90:
							position, tokenIndex, depth = position89, tokenIndex89, depth89
							{
								position91, tokenIndex91, depth91 := position, tokenIndex, depth
								if !_rules[ruleEOL]() {
									goto l91
								}
								goto l88
							l91:
								position, tokenIndex, depth = position91, tokenIndex91, depth91
							}
							{
								position92, tokenIndex92, depth92 := position, tokenIndex, depth
								if buffer[position] != rune('\'') {
									goto l92
								}
								position++
								goto l88
							l92:
								position, tokenIndex, depth = position92, tokenIndex92, depth92
							}
							if !matchDot() {
								goto l88
							}
						}
					l89:
						goto l87
					l88:
						position, tokenIndex, depth = position88, tokenIndex88, depth88
					}
					if buffer[position] != rune('\'') {
						goto l84
					}
					position++
					depth--
					add(rulePegText, position86)
				}
				if !_rules[ruleAction5]() {
					goto l84
				}
				depth--
				add(ruleSingleQuotedString, position85)
			}
			return true
		l84:
			position, tokenIndex, depth = position84, tokenIndex84, depth84
			return false
		},
		/* 12 DoubleQuotedString <- <(<('"' (('\\' '"') / (!EOL !'"' .))* '"')> Action6)> */
		func() bool {
			position93, tokenIndex93, depth93 := position, tokenIndex, depth
			{
				position94 := position
				depth++
				{
					position95 := position
					depth++
					if buffer[position] != rune('"') {
						goto l93
					}
					position++
				l96:
					{
						position97, tokenIndex97, depth97 := position, tokenIndex, depth
						{
							position98, tokenIndex98, depth98 := position, tokenIndex, depth
							if buffer[position] != rune('\\') {
								goto l99
							}
							position++
							if buffer[position] != rune('"') {
								goto l99
							}
							position++
							goto l98
						l99:
							position, tokenIndex, depth = position98, tokenIndex98, depth98
							{
								position100, tokenIndex100, depth100 := position, tokenIndex, depth
								if !_rules[ruleEOL]() {
									goto l100
								}
								goto l97
							l100:
								position, tokenIndex, depth = position100, tokenIndex100, depth100
							}
							{
								position101, tokenIndex101, depth101 := position, tokenIndex, depth
								if buffer[position] != rune('"') {
									goto l101
								}
								position++
								goto l97
							l101:
								position, tokenIndex, depth = position101, tokenIndex101, depth101
							}
							if !matchDot() {
								goto l97
							}
						}
					l98:
						goto l96
					l97:
						position, tokenIndex, depth = position97, tokenIndex97, depth97
					}
					if buffer[position] != rune('"') {
						goto l93
					}
					position++
					depth--
					add(rulePegText, position95)
				}
				if !_rules[ruleAction6]() {
					goto l93
				}
				depth--
				add(ruleDoubleQuotedString, position94)
			}
			return true
		l93:
			position, tokenIndex, depth = position93, tokenIndex93, depth93
			return false
		},
		/* 13 RawString <- <(<('`' (!'`' .)* '`')> Action7)> */
		func() bool {
			position102, tokenIndex102, depth102 := position, tokenIndex, depth
			{
				position103 := position
				depth++
				{
					position104 := position
					depth++
					if buffer[position] != rune('`') {
						goto l102
					}
					position++
				l105:
					{
						position106, tokenIndex106, depth106 := position, tokenIndex, depth
						{
							position107, tokenIndex107, depth107 := position, tokenIndex, depth
							if buffer[position] != rune('`') {
								goto l107
							}
							position++
							goto l106
						l107:
							position, tokenIndex, depth = position107, tokenIndex107, depth107
						}
						if !matchDot() {
							goto l106
						}
						goto l105
					l106:
						position, tokenIndex, depth = position106, tokenIndex106, depth106
					}
					if buffer[position] != rune('`') {
						goto l102
					}
					position++
					depth--
					add(rulePegText, position104)
				}
				if !_rules[ruleAction7]() {
					goto l102
				}
				depth--
				add(ruleRawString, position103)
			}
			return true
		l102:
			position, tokenIndex, depth = position102, tokenIndex102, depth102
			return false
		},
		/* 14 Comment <- <(s '#' (!EOL .)*)> */
		func() bool {
			position108, tokenIndex108, depth108 := position, tokenIndex, depth
			{
				position109 := position
				depth++
				if !_rules[rules]() {
					goto l108
				}
				if buffer[position] != rune('#') {
					goto l108
				}
				position++
			l110:
				{
					position111, tokenIndex111, depth111 := position, tokenIndex, depth
					{
						position112, tokenIndex112, depth112 := position, tokenIndex, depth
						if !_rules[ruleEOL]() {
							goto l112
						}
						goto l111
					l112:
						position, tokenIndex, depth = position112, tokenIndex112, depth112
					}
					if !matchDot() {
						goto l111
					}
					goto l110
				l111:
					position, tokenIndex, depth = position111, tokenIndex111, depth111
				}
				depth--
				add(ruleComment, position109)
			}
			return true
		l108:
			position, tokenIndex, depth = position108, tokenIndex108, depth108
			return false
		},
		/* 15 EOF <- <!.> */
		func() bool {
			position113, tokenIndex113, depth113 := position, tokenIndex, depth
			{
				position114 := position
				depth++
				{
					position115, tokenIndex115, depth115 := position, tokenIndex, depth
					if !matchDot() {
						goto l115
					}
					goto l113
				l115:
					position, tokenIndex, depth = position115, tokenIndex115, depth115
				}
				depth--
				add(ruleEOF, position114)
			}
			return true
		l113:
			position, tokenIndex, depth = position113, tokenIndex113, depth113
			return false
		},
		/* 16 EOL <- <('\r' / '\n')> */
		func() bool {
			position116, tokenIndex116, depth116 := position, tokenIndex, depth
			{
				position117 := position
				depth++
				{
					position118, tokenIndex118, depth118 := position, tokenIndex, depth
					if buffer[position] != rune('\r') {
						goto l119
					}
					position++
					goto l118
				l119:
					position, tokenIndex, depth = position118, tokenIndex118, depth118
					if buffer[position] != rune('\n') {
						goto l116
					}
					position++
				}
			l118:
				depth--
				add(ruleEOL, position117)
			}
			return true
		l116:
			position, tokenIndex, depth = position116, tokenIndex116, depth116
			return false
		},
		/* 17 s <- <(' ' / '\t')*> */
		func() bool {
			{
				position121 := position
				depth++
			l122:
				{
					position123, tokenIndex123, depth123 := position, tokenIndex, depth
					{
						position124, tokenIndex124, depth124 := position, tokenIndex, depth
						if buffer[position] != rune(' ') {
							goto l125
						}
						position++
						goto l124
					l125:
						position, tokenIndex, depth = position124, tokenIndex124, depth124
						if buffer[position] != rune('\t') {
							goto l123
						}
						position++
					}
				l124:
					goto l122
				l123:
					position, tokenIndex, depth = position123, tokenIndex123, depth123
				}
				depth--
				add(rules, position121)
			}
			return true
		},
		nil,
		/* 20 Action0 <- <{ p.name = buffer[begin:end] }> */
		func() bool {
			{
				add(ruleAction0, position)
			}
			return true
		},
		/* 21 Action1 <- <{ p.set(p.name, p.value) }> */
		func() bool {
			{
				add(ruleAction1, position)
			}
			return true
		},
		/* 22 Action2 <- <{ p.inField = false }> */
		func() bool {
			{
				add(ruleAction2, position)
			}
			return true
		},
		/* 23 Action3 <- <{ p.inField = true; p.newField(buffer[begin:end]) }> */
		func() bool {
			{
				add(ruleAction3, position)
			}
			return true
		},
		/* 24 Action4 <- <{ p.value = buffer[begin:end] }> */
		func() bool {
			{
				add(ruleAction4, position)
			}
			return true
		},
		/* 25 Action5 <- <{ p.value = buffer[begin:end] }> */
		func() bool {
			{
				add(ruleAction5, position)
			}
			return true
		},
		/* 26 Action6 <- <{ p.value = buffer[begin:end] }> */
		func() bool {
			{
				add(ruleAction6, position)
			}
			return true
		},
		/* 27 Action7 <- <{ p.value = buffer[begin:end] }> */
		func() bool {
			{
				add(ruleAction7, position)
			}
			return true
		},
	}
	p.rules = _rules
}
