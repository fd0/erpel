package rules

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
	ruleSeparator
	ruleTemplates
	ruleTemplate
	ruleSamples
	ruleSample
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
	ruleAction8
	ruleAction9

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
	"Separator",
	"Templates",
	"Template",
	"Samples",
	"Sample",
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
	"Action8",
	"Action9",

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

type ruleParser struct {
	State

	Buffer string
	buffer []rune
	rules  [35]func() bool
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
	p   *ruleParser
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

func (p *ruleParser) PrintSyntaxTree() {
	p.tokenTree.PrintSyntaxTree(p.Buffer)
}

func (p *ruleParser) Highlighter() {
	p.tokenTree.PrintSyntax()
}

func (p *ruleParser) Execute() {
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
		case ruleAction8:
			p.addTemplate(buffer[begin:end])
		case ruleAction9:
			p.addSample(buffer[begin:end])

		}
	}
	_, _, _, _, _ = buffer, _buffer, text, begin, end
}

func (p *ruleParser) Init() {
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
		/* 0 start <- <((Line EOL)* Line? (Separator Templates (Separator Samples)?)? EOF)> */
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
				{
					position6, tokenIndex6, depth6 := position, tokenIndex, depth
					if !_rules[ruleSeparator]() {
						goto l6
					}
					if !_rules[ruleTemplates]() {
						goto l6
					}
					{
						position8, tokenIndex8, depth8 := position, tokenIndex, depth
						if !_rules[ruleSeparator]() {
							goto l8
						}
						if !_rules[ruleSamples]() {
							goto l8
						}
						goto l9
					l8:
						position, tokenIndex, depth = position8, tokenIndex8, depth8
					}
				l9:
					goto l7
				l6:
					position, tokenIndex, depth = position6, tokenIndex6, depth6
				}
			l7:
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
			position10, tokenIndex10, depth10 := position, tokenIndex, depth
			{
				position11 := position
				depth++
				{
					position12, tokenIndex12, depth12 := position, tokenIndex, depth
					{
						position14, tokenIndex14, depth14 := position, tokenIndex, depth
						if !_rules[ruleField]() {
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
				{
					position16, tokenIndex16, depth16 := position, tokenIndex, depth
					if !_rules[ruleComment]() {
						goto l16
					}
					goto l17
				l16:
					position, tokenIndex, depth = position16, tokenIndex16, depth16
				}
			l17:
				depth--
				add(ruleLine, position11)
			}
			return true
		l10:
			position, tokenIndex, depth = position10, tokenIndex10, depth10
			return false
		},
		/* 2 Name <- <(<([a-z] / [A-Z] / [0-9] / '-' / '_')+> Action0)> */
		func() bool {
			position18, tokenIndex18, depth18 := position, tokenIndex, depth
			{
				position19 := position
				depth++
				{
					position20 := position
					depth++
					{
						position23, tokenIndex23, depth23 := position, tokenIndex, depth
						if c := buffer[position]; c < rune('a') || c > rune('z') {
							goto l24
						}
						position++
						goto l23
					l24:
						position, tokenIndex, depth = position23, tokenIndex23, depth23
						if c := buffer[position]; c < rune('A') || c > rune('Z') {
							goto l25
						}
						position++
						goto l23
					l25:
						position, tokenIndex, depth = position23, tokenIndex23, depth23
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l26
						}
						position++
						goto l23
					l26:
						position, tokenIndex, depth = position23, tokenIndex23, depth23
						if buffer[position] != rune('-') {
							goto l27
						}
						position++
						goto l23
					l27:
						position, tokenIndex, depth = position23, tokenIndex23, depth23
						if buffer[position] != rune('_') {
							goto l18
						}
						position++
					}
				l23:
				l21:
					{
						position22, tokenIndex22, depth22 := position, tokenIndex, depth
						{
							position28, tokenIndex28, depth28 := position, tokenIndex, depth
							if c := buffer[position]; c < rune('a') || c > rune('z') {
								goto l29
							}
							position++
							goto l28
						l29:
							position, tokenIndex, depth = position28, tokenIndex28, depth28
							if c := buffer[position]; c < rune('A') || c > rune('Z') {
								goto l30
							}
							position++
							goto l28
						l30:
							position, tokenIndex, depth = position28, tokenIndex28, depth28
							if c := buffer[position]; c < rune('0') || c > rune('9') {
								goto l31
							}
							position++
							goto l28
						l31:
							position, tokenIndex, depth = position28, tokenIndex28, depth28
							if buffer[position] != rune('-') {
								goto l32
							}
							position++
							goto l28
						l32:
							position, tokenIndex, depth = position28, tokenIndex28, depth28
							if buffer[position] != rune('_') {
								goto l22
							}
							position++
						}
					l28:
						goto l21
					l22:
						position, tokenIndex, depth = position22, tokenIndex22, depth22
					}
					depth--
					add(rulePegText, position20)
				}
				if !_rules[ruleAction0]() {
					goto l18
				}
				depth--
				add(ruleName, position19)
			}
			return true
		l18:
			position, tokenIndex, depth = position18, tokenIndex18, depth18
			return false
		},
		/* 3 Statement <- <(s Name s '=' s Value Action1)> */
		func() bool {
			position33, tokenIndex33, depth33 := position, tokenIndex, depth
			{
				position34 := position
				depth++
				if !_rules[rules]() {
					goto l33
				}
				if !_rules[ruleName]() {
					goto l33
				}
				if !_rules[rules]() {
					goto l33
				}
				if buffer[position] != rune('=') {
					goto l33
				}
				position++
				if !_rules[rules]() {
					goto l33
				}
				if !_rules[ruleValue]() {
					goto l33
				}
				if !_rules[ruleAction1]() {
					goto l33
				}
				depth--
				add(ruleStatement, position34)
			}
			return true
		l33:
			position, tokenIndex, depth = position33, tokenIndex33, depth33
			return false
		},
		/* 4 Field <- <(s (('f' / 'F') ('i' / 'I') ('e' / 'E') ('l' / 'L') ('d' / 'D')) s FieldName s '{' FieldData '}' Action2)> */
		func() bool {
			position35, tokenIndex35, depth35 := position, tokenIndex, depth
			{
				position36 := position
				depth++
				if !_rules[rules]() {
					goto l35
				}
				{
					position37, tokenIndex37, depth37 := position, tokenIndex, depth
					if buffer[position] != rune('f') {
						goto l38
					}
					position++
					goto l37
				l38:
					position, tokenIndex, depth = position37, tokenIndex37, depth37
					if buffer[position] != rune('F') {
						goto l35
					}
					position++
				}
			l37:
				{
					position39, tokenIndex39, depth39 := position, tokenIndex, depth
					if buffer[position] != rune('i') {
						goto l40
					}
					position++
					goto l39
				l40:
					position, tokenIndex, depth = position39, tokenIndex39, depth39
					if buffer[position] != rune('I') {
						goto l35
					}
					position++
				}
			l39:
				{
					position41, tokenIndex41, depth41 := position, tokenIndex, depth
					if buffer[position] != rune('e') {
						goto l42
					}
					position++
					goto l41
				l42:
					position, tokenIndex, depth = position41, tokenIndex41, depth41
					if buffer[position] != rune('E') {
						goto l35
					}
					position++
				}
			l41:
				{
					position43, tokenIndex43, depth43 := position, tokenIndex, depth
					if buffer[position] != rune('l') {
						goto l44
					}
					position++
					goto l43
				l44:
					position, tokenIndex, depth = position43, tokenIndex43, depth43
					if buffer[position] != rune('L') {
						goto l35
					}
					position++
				}
			l43:
				{
					position45, tokenIndex45, depth45 := position, tokenIndex, depth
					if buffer[position] != rune('d') {
						goto l46
					}
					position++
					goto l45
				l46:
					position, tokenIndex, depth = position45, tokenIndex45, depth45
					if buffer[position] != rune('D') {
						goto l35
					}
					position++
				}
			l45:
				if !_rules[rules]() {
					goto l35
				}
				if !_rules[ruleFieldName]() {
					goto l35
				}
				if !_rules[rules]() {
					goto l35
				}
				if buffer[position] != rune('{') {
					goto l35
				}
				position++
				if !_rules[ruleFieldData]() {
					goto l35
				}
				if buffer[position] != rune('}') {
					goto l35
				}
				position++
				if !_rules[ruleAction2]() {
					goto l35
				}
				depth--
				add(ruleField, position36)
			}
			return true
		l35:
			position, tokenIndex, depth = position35, tokenIndex35, depth35
			return false
		},
		/* 5 FieldName <- <(<([a-z] / [A-Z] / [0-9] / '-' / '_')+> Action3)> */
		func() bool {
			position47, tokenIndex47, depth47 := position, tokenIndex, depth
			{
				position48 := position
				depth++
				{
					position49 := position
					depth++
					{
						position52, tokenIndex52, depth52 := position, tokenIndex, depth
						if c := buffer[position]; c < rune('a') || c > rune('z') {
							goto l53
						}
						position++
						goto l52
					l53:
						position, tokenIndex, depth = position52, tokenIndex52, depth52
						if c := buffer[position]; c < rune('A') || c > rune('Z') {
							goto l54
						}
						position++
						goto l52
					l54:
						position, tokenIndex, depth = position52, tokenIndex52, depth52
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l55
						}
						position++
						goto l52
					l55:
						position, tokenIndex, depth = position52, tokenIndex52, depth52
						if buffer[position] != rune('-') {
							goto l56
						}
						position++
						goto l52
					l56:
						position, tokenIndex, depth = position52, tokenIndex52, depth52
						if buffer[position] != rune('_') {
							goto l47
						}
						position++
					}
				l52:
				l50:
					{
						position51, tokenIndex51, depth51 := position, tokenIndex, depth
						{
							position57, tokenIndex57, depth57 := position, tokenIndex, depth
							if c := buffer[position]; c < rune('a') || c > rune('z') {
								goto l58
							}
							position++
							goto l57
						l58:
							position, tokenIndex, depth = position57, tokenIndex57, depth57
							if c := buffer[position]; c < rune('A') || c > rune('Z') {
								goto l59
							}
							position++
							goto l57
						l59:
							position, tokenIndex, depth = position57, tokenIndex57, depth57
							if c := buffer[position]; c < rune('0') || c > rune('9') {
								goto l60
							}
							position++
							goto l57
						l60:
							position, tokenIndex, depth = position57, tokenIndex57, depth57
							if buffer[position] != rune('-') {
								goto l61
							}
							position++
							goto l57
						l61:
							position, tokenIndex, depth = position57, tokenIndex57, depth57
							if buffer[position] != rune('_') {
								goto l51
							}
							position++
						}
					l57:
						goto l50
					l51:
						position, tokenIndex, depth = position51, tokenIndex51, depth51
					}
					depth--
					add(rulePegText, position49)
				}
				if !_rules[ruleAction3]() {
					goto l47
				}
				depth--
				add(ruleFieldName, position48)
			}
			return true
		l47:
			position, tokenIndex, depth = position47, tokenIndex47, depth47
			return false
		},
		/* 6 FieldData <- <((FieldStatement EOL)* FieldStatement?)> */
		func() bool {
			{
				position63 := position
				depth++
			l64:
				{
					position65, tokenIndex65, depth65 := position, tokenIndex, depth
					if !_rules[ruleFieldStatement]() {
						goto l65
					}
					if !_rules[ruleEOL]() {
						goto l65
					}
					goto l64
				l65:
					position, tokenIndex, depth = position65, tokenIndex65, depth65
				}
				{
					position66, tokenIndex66, depth66 := position, tokenIndex, depth
					if !_rules[ruleFieldStatement]() {
						goto l66
					}
					goto l67
				l66:
					position, tokenIndex, depth = position66, tokenIndex66, depth66
				}
			l67:
				depth--
				add(ruleFieldData, position63)
			}
			return true
		},
		/* 7 FieldStatement <- <(Statement? s Comment?)> */
		func() bool {
			position68, tokenIndex68, depth68 := position, tokenIndex, depth
			{
				position69 := position
				depth++
				{
					position70, tokenIndex70, depth70 := position, tokenIndex, depth
					if !_rules[ruleStatement]() {
						goto l70
					}
					goto l71
				l70:
					position, tokenIndex, depth = position70, tokenIndex70, depth70
				}
			l71:
				if !_rules[rules]() {
					goto l68
				}
				{
					position72, tokenIndex72, depth72 := position, tokenIndex, depth
					if !_rules[ruleComment]() {
						goto l72
					}
					goto l73
				l72:
					position, tokenIndex, depth = position72, tokenIndex72, depth72
				}
			l73:
				depth--
				add(ruleFieldStatement, position69)
			}
			return true
		l68:
			position, tokenIndex, depth = position68, tokenIndex68, depth68
			return false
		},
		/* 8 Value <- <(List / String)> */
		func() bool {
			position74, tokenIndex74, depth74 := position, tokenIndex, depth
			{
				position75 := position
				depth++
				{
					position76, tokenIndex76, depth76 := position, tokenIndex, depth
					if !_rules[ruleList]() {
						goto l77
					}
					goto l76
				l77:
					position, tokenIndex, depth = position76, tokenIndex76, depth76
					if !_rules[ruleString]() {
						goto l74
					}
				}
			l76:
				depth--
				add(ruleValue, position75)
			}
			return true
		l74:
			position, tokenIndex, depth = position74, tokenIndex74, depth74
			return false
		},
		/* 9 String <- <(DoubleQuotedString / SingleQuotedString / RawString)> */
		func() bool {
			position78, tokenIndex78, depth78 := position, tokenIndex, depth
			{
				position79 := position
				depth++
				{
					position80, tokenIndex80, depth80 := position, tokenIndex, depth
					if !_rules[ruleDoubleQuotedString]() {
						goto l81
					}
					goto l80
				l81:
					position, tokenIndex, depth = position80, tokenIndex80, depth80
					if !_rules[ruleSingleQuotedString]() {
						goto l82
					}
					goto l80
				l82:
					position, tokenIndex, depth = position80, tokenIndex80, depth80
					if !_rules[ruleRawString]() {
						goto l78
					}
				}
			l80:
				depth--
				add(ruleString, position79)
			}
			return true
		l78:
			position, tokenIndex, depth = position78, tokenIndex78, depth78
			return false
		},
		/* 10 List <- <(<('[' s (s String s ',' s)* s String s ']')> Action4)> */
		func() bool {
			position83, tokenIndex83, depth83 := position, tokenIndex, depth
			{
				position84 := position
				depth++
				{
					position85 := position
					depth++
					if buffer[position] != rune('[') {
						goto l83
					}
					position++
					if !_rules[rules]() {
						goto l83
					}
				l86:
					{
						position87, tokenIndex87, depth87 := position, tokenIndex, depth
						if !_rules[rules]() {
							goto l87
						}
						if !_rules[ruleString]() {
							goto l87
						}
						if !_rules[rules]() {
							goto l87
						}
						if buffer[position] != rune(',') {
							goto l87
						}
						position++
						if !_rules[rules]() {
							goto l87
						}
						goto l86
					l87:
						position, tokenIndex, depth = position87, tokenIndex87, depth87
					}
					if !_rules[rules]() {
						goto l83
					}
					if !_rules[ruleString]() {
						goto l83
					}
					if !_rules[rules]() {
						goto l83
					}
					if buffer[position] != rune(']') {
						goto l83
					}
					position++
					depth--
					add(rulePegText, position85)
				}
				if !_rules[ruleAction4]() {
					goto l83
				}
				depth--
				add(ruleList, position84)
			}
			return true
		l83:
			position, tokenIndex, depth = position83, tokenIndex83, depth83
			return false
		},
		/* 11 SingleQuotedString <- <(<('\'' (('\\' '\'') / (!EOL !'\'' .))* '\'')> Action5)> */
		func() bool {
			position88, tokenIndex88, depth88 := position, tokenIndex, depth
			{
				position89 := position
				depth++
				{
					position90 := position
					depth++
					if buffer[position] != rune('\'') {
						goto l88
					}
					position++
				l91:
					{
						position92, tokenIndex92, depth92 := position, tokenIndex, depth
						{
							position93, tokenIndex93, depth93 := position, tokenIndex, depth
							if buffer[position] != rune('\\') {
								goto l94
							}
							position++
							if buffer[position] != rune('\'') {
								goto l94
							}
							position++
							goto l93
						l94:
							position, tokenIndex, depth = position93, tokenIndex93, depth93
							{
								position95, tokenIndex95, depth95 := position, tokenIndex, depth
								if !_rules[ruleEOL]() {
									goto l95
								}
								goto l92
							l95:
								position, tokenIndex, depth = position95, tokenIndex95, depth95
							}
							{
								position96, tokenIndex96, depth96 := position, tokenIndex, depth
								if buffer[position] != rune('\'') {
									goto l96
								}
								position++
								goto l92
							l96:
								position, tokenIndex, depth = position96, tokenIndex96, depth96
							}
							if !matchDot() {
								goto l92
							}
						}
					l93:
						goto l91
					l92:
						position, tokenIndex, depth = position92, tokenIndex92, depth92
					}
					if buffer[position] != rune('\'') {
						goto l88
					}
					position++
					depth--
					add(rulePegText, position90)
				}
				if !_rules[ruleAction5]() {
					goto l88
				}
				depth--
				add(ruleSingleQuotedString, position89)
			}
			return true
		l88:
			position, tokenIndex, depth = position88, tokenIndex88, depth88
			return false
		},
		/* 12 DoubleQuotedString <- <(<('"' (('\\' '"') / (!EOL !'"' .))* '"')> Action6)> */
		func() bool {
			position97, tokenIndex97, depth97 := position, tokenIndex, depth
			{
				position98 := position
				depth++
				{
					position99 := position
					depth++
					if buffer[position] != rune('"') {
						goto l97
					}
					position++
				l100:
					{
						position101, tokenIndex101, depth101 := position, tokenIndex, depth
						{
							position102, tokenIndex102, depth102 := position, tokenIndex, depth
							if buffer[position] != rune('\\') {
								goto l103
							}
							position++
							if buffer[position] != rune('"') {
								goto l103
							}
							position++
							goto l102
						l103:
							position, tokenIndex, depth = position102, tokenIndex102, depth102
							{
								position104, tokenIndex104, depth104 := position, tokenIndex, depth
								if !_rules[ruleEOL]() {
									goto l104
								}
								goto l101
							l104:
								position, tokenIndex, depth = position104, tokenIndex104, depth104
							}
							{
								position105, tokenIndex105, depth105 := position, tokenIndex, depth
								if buffer[position] != rune('"') {
									goto l105
								}
								position++
								goto l101
							l105:
								position, tokenIndex, depth = position105, tokenIndex105, depth105
							}
							if !matchDot() {
								goto l101
							}
						}
					l102:
						goto l100
					l101:
						position, tokenIndex, depth = position101, tokenIndex101, depth101
					}
					if buffer[position] != rune('"') {
						goto l97
					}
					position++
					depth--
					add(rulePegText, position99)
				}
				if !_rules[ruleAction6]() {
					goto l97
				}
				depth--
				add(ruleDoubleQuotedString, position98)
			}
			return true
		l97:
			position, tokenIndex, depth = position97, tokenIndex97, depth97
			return false
		},
		/* 13 RawString <- <(<('`' (!'`' .)* '`')> Action7)> */
		func() bool {
			position106, tokenIndex106, depth106 := position, tokenIndex, depth
			{
				position107 := position
				depth++
				{
					position108 := position
					depth++
					if buffer[position] != rune('`') {
						goto l106
					}
					position++
				l109:
					{
						position110, tokenIndex110, depth110 := position, tokenIndex, depth
						{
							position111, tokenIndex111, depth111 := position, tokenIndex, depth
							if buffer[position] != rune('`') {
								goto l111
							}
							position++
							goto l110
						l111:
							position, tokenIndex, depth = position111, tokenIndex111, depth111
						}
						if !matchDot() {
							goto l110
						}
						goto l109
					l110:
						position, tokenIndex, depth = position110, tokenIndex110, depth110
					}
					if buffer[position] != rune('`') {
						goto l106
					}
					position++
					depth--
					add(rulePegText, position108)
				}
				if !_rules[ruleAction7]() {
					goto l106
				}
				depth--
				add(ruleRawString, position107)
			}
			return true
		l106:
			position, tokenIndex, depth = position106, tokenIndex106, depth106
			return false
		},
		/* 14 Separator <- <(s ('-' '-' '-') '-'* s EOL)> */
		func() bool {
			position112, tokenIndex112, depth112 := position, tokenIndex, depth
			{
				position113 := position
				depth++
				if !_rules[rules]() {
					goto l112
				}
				if buffer[position] != rune('-') {
					goto l112
				}
				position++
				if buffer[position] != rune('-') {
					goto l112
				}
				position++
				if buffer[position] != rune('-') {
					goto l112
				}
				position++
			l114:
				{
					position115, tokenIndex115, depth115 := position, tokenIndex, depth
					if buffer[position] != rune('-') {
						goto l115
					}
					position++
					goto l114
				l115:
					position, tokenIndex, depth = position115, tokenIndex115, depth115
				}
				if !_rules[rules]() {
					goto l112
				}
				if !_rules[ruleEOL]() {
					goto l112
				}
				depth--
				add(ruleSeparator, position113)
			}
			return true
		l112:
			position, tokenIndex, depth = position112, tokenIndex112, depth112
			return false
		},
		/* 15 Templates <- <(!Separator (Comment / Template) EOL)*> */
		func() bool {
			{
				position117 := position
				depth++
			l118:
				{
					position119, tokenIndex119, depth119 := position, tokenIndex, depth
					{
						position120, tokenIndex120, depth120 := position, tokenIndex, depth
						if !_rules[ruleSeparator]() {
							goto l120
						}
						goto l119
					l120:
						position, tokenIndex, depth = position120, tokenIndex120, depth120
					}
					{
						position121, tokenIndex121, depth121 := position, tokenIndex, depth
						if !_rules[ruleComment]() {
							goto l122
						}
						goto l121
					l122:
						position, tokenIndex, depth = position121, tokenIndex121, depth121
						if !_rules[ruleTemplate]() {
							goto l119
						}
					}
				l121:
					if !_rules[ruleEOL]() {
						goto l119
					}
					goto l118
				l119:
					position, tokenIndex, depth = position119, tokenIndex119, depth119
				}
				depth--
				add(ruleTemplates, position117)
			}
			return true
		},
		/* 16 Template <- <(s <(!EOL .)*> Action8)> */
		func() bool {
			position123, tokenIndex123, depth123 := position, tokenIndex, depth
			{
				position124 := position
				depth++
				if !_rules[rules]() {
					goto l123
				}
				{
					position125 := position
					depth++
				l126:
					{
						position127, tokenIndex127, depth127 := position, tokenIndex, depth
						{
							position128, tokenIndex128, depth128 := position, tokenIndex, depth
							if !_rules[ruleEOL]() {
								goto l128
							}
							goto l127
						l128:
							position, tokenIndex, depth = position128, tokenIndex128, depth128
						}
						if !matchDot() {
							goto l127
						}
						goto l126
					l127:
						position, tokenIndex, depth = position127, tokenIndex127, depth127
					}
					depth--
					add(rulePegText, position125)
				}
				if !_rules[ruleAction8]() {
					goto l123
				}
				depth--
				add(ruleTemplate, position124)
			}
			return true
		l123:
			position, tokenIndex, depth = position123, tokenIndex123, depth123
			return false
		},
		/* 17 Samples <- <((Comment / Sample) EOL)*> */
		func() bool {
			{
				position130 := position
				depth++
			l131:
				{
					position132, tokenIndex132, depth132 := position, tokenIndex, depth
					{
						position133, tokenIndex133, depth133 := position, tokenIndex, depth
						if !_rules[ruleComment]() {
							goto l134
						}
						goto l133
					l134:
						position, tokenIndex, depth = position133, tokenIndex133, depth133
						if !_rules[ruleSample]() {
							goto l132
						}
					}
				l133:
					if !_rules[ruleEOL]() {
						goto l132
					}
					goto l131
				l132:
					position, tokenIndex, depth = position132, tokenIndex132, depth132
				}
				depth--
				add(ruleSamples, position130)
			}
			return true
		},
		/* 18 Sample <- <(s <(!EOL .)*> Action9)> */
		func() bool {
			position135, tokenIndex135, depth135 := position, tokenIndex, depth
			{
				position136 := position
				depth++
				if !_rules[rules]() {
					goto l135
				}
				{
					position137 := position
					depth++
				l138:
					{
						position139, tokenIndex139, depth139 := position, tokenIndex, depth
						{
							position140, tokenIndex140, depth140 := position, tokenIndex, depth
							if !_rules[ruleEOL]() {
								goto l140
							}
							goto l139
						l140:
							position, tokenIndex, depth = position140, tokenIndex140, depth140
						}
						if !matchDot() {
							goto l139
						}
						goto l138
					l139:
						position, tokenIndex, depth = position139, tokenIndex139, depth139
					}
					depth--
					add(rulePegText, position137)
				}
				if !_rules[ruleAction9]() {
					goto l135
				}
				depth--
				add(ruleSample, position136)
			}
			return true
		l135:
			position, tokenIndex, depth = position135, tokenIndex135, depth135
			return false
		},
		/* 19 Comment <- <(s '#' (!EOL .)*)> */
		func() bool {
			position141, tokenIndex141, depth141 := position, tokenIndex, depth
			{
				position142 := position
				depth++
				if !_rules[rules]() {
					goto l141
				}
				if buffer[position] != rune('#') {
					goto l141
				}
				position++
			l143:
				{
					position144, tokenIndex144, depth144 := position, tokenIndex, depth
					{
						position145, tokenIndex145, depth145 := position, tokenIndex, depth
						if !_rules[ruleEOL]() {
							goto l145
						}
						goto l144
					l145:
						position, tokenIndex, depth = position145, tokenIndex145, depth145
					}
					if !matchDot() {
						goto l144
					}
					goto l143
				l144:
					position, tokenIndex, depth = position144, tokenIndex144, depth144
				}
				depth--
				add(ruleComment, position142)
			}
			return true
		l141:
			position, tokenIndex, depth = position141, tokenIndex141, depth141
			return false
		},
		/* 20 EOF <- <!.> */
		func() bool {
			position146, tokenIndex146, depth146 := position, tokenIndex, depth
			{
				position147 := position
				depth++
				{
					position148, tokenIndex148, depth148 := position, tokenIndex, depth
					if !matchDot() {
						goto l148
					}
					goto l146
				l148:
					position, tokenIndex, depth = position148, tokenIndex148, depth148
				}
				depth--
				add(ruleEOF, position147)
			}
			return true
		l146:
			position, tokenIndex, depth = position146, tokenIndex146, depth146
			return false
		},
		/* 21 EOL <- <('\r' / '\n')> */
		func() bool {
			position149, tokenIndex149, depth149 := position, tokenIndex, depth
			{
				position150 := position
				depth++
				{
					position151, tokenIndex151, depth151 := position, tokenIndex, depth
					if buffer[position] != rune('\r') {
						goto l152
					}
					position++
					goto l151
				l152:
					position, tokenIndex, depth = position151, tokenIndex151, depth151
					if buffer[position] != rune('\n') {
						goto l149
					}
					position++
				}
			l151:
				depth--
				add(ruleEOL, position150)
			}
			return true
		l149:
			position, tokenIndex, depth = position149, tokenIndex149, depth149
			return false
		},
		/* 22 s <- <(' ' / '\t')*> */
		func() bool {
			{
				position154 := position
				depth++
			l155:
				{
					position156, tokenIndex156, depth156 := position, tokenIndex, depth
					{
						position157, tokenIndex157, depth157 := position, tokenIndex, depth
						if buffer[position] != rune(' ') {
							goto l158
						}
						position++
						goto l157
					l158:
						position, tokenIndex, depth = position157, tokenIndex157, depth157
						if buffer[position] != rune('\t') {
							goto l156
						}
						position++
					}
				l157:
					goto l155
				l156:
					position, tokenIndex, depth = position156, tokenIndex156, depth156
				}
				depth--
				add(rules, position154)
			}
			return true
		},
		nil,
		/* 25 Action0 <- <{ p.name = buffer[begin:end] }> */
		func() bool {
			{
				add(ruleAction0, position)
			}
			return true
		},
		/* 26 Action1 <- <{ p.set(p.name, p.value) }> */
		func() bool {
			{
				add(ruleAction1, position)
			}
			return true
		},
		/* 27 Action2 <- <{ p.inField = false }> */
		func() bool {
			{
				add(ruleAction2, position)
			}
			return true
		},
		/* 28 Action3 <- <{ p.inField = true; p.newField(buffer[begin:end]) }> */
		func() bool {
			{
				add(ruleAction3, position)
			}
			return true
		},
		/* 29 Action4 <- <{ p.value = buffer[begin:end] }> */
		func() bool {
			{
				add(ruleAction4, position)
			}
			return true
		},
		/* 30 Action5 <- <{ p.value = buffer[begin:end] }> */
		func() bool {
			{
				add(ruleAction5, position)
			}
			return true
		},
		/* 31 Action6 <- <{ p.value = buffer[begin:end] }> */
		func() bool {
			{
				add(ruleAction6, position)
			}
			return true
		},
		/* 32 Action7 <- <{ p.value = buffer[begin:end] }> */
		func() bool {
			{
				add(ruleAction7, position)
			}
			return true
		},
		/* 33 Action8 <- <{ p.addTemplate(buffer[begin:end]) }> */
		func() bool {
			{
				add(ruleAction8, position)
			}
			return true
		},
		/* 34 Action9 <- <{ p.addSample(buffer[begin:end]) }> */
		func() bool {
			{
				add(ruleAction9, position)
			}
			return true
		},
	}
	p.rules = _rules
}
