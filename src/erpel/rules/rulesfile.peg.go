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
	ruleField
	ruleFieldName
	rulestatements
	ruleline
	ruleName
	ruleStatement
	ruleString
	ruleSingleQuotedString
	ruleDoubleQuotedString
	ruleRawString
	ruleComment
	ruleEOF
	ruleEOL
	rules
	ruleS
	rulePegText
	ruleAction0
	ruleAction1
	ruleAction2
	ruleAction3
	ruleAction4
	ruleAction5

	rulePre
	ruleIn
	ruleSuf
)

var rul3s = [...]string{
	"Unknown",
	"start",
	"Field",
	"FieldName",
	"statements",
	"line",
	"Name",
	"Statement",
	"String",
	"SingleQuotedString",
	"DoubleQuotedString",
	"RawString",
	"Comment",
	"EOF",
	"EOL",
	"s",
	"S",
	"PegText",
	"Action0",
	"Action1",
	"Action2",
	"Action3",
	"Action4",
	"Action5",

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
	ruleState

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
			p.newField(buffer[begin:end])
		case ruleAction1:
			p.name = buffer[begin:end]
		case ruleAction2:
			p.set(p.name, p.value)
		case ruleAction3:
			p.value = buffer[begin:end]
		case ruleAction4:
			p.value = buffer[begin:end]
		case ruleAction5:
			p.value = buffer[begin:end]

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
		/* 0 start <- <(S (Field EOL?)* EOF)> */
		func() bool {
			position0, tokenIndex0, depth0 := position, tokenIndex, depth
			{
				position1 := position
				depth++
				if !_rules[ruleS]() {
					goto l0
				}
			l2:
				{
					position3, tokenIndex3, depth3 := position, tokenIndex, depth
					if !_rules[ruleField]() {
						goto l3
					}
					{
						position4, tokenIndex4, depth4 := position, tokenIndex, depth
						if !_rules[ruleEOL]() {
							goto l4
						}
						goto l5
					l4:
						position, tokenIndex, depth = position4, tokenIndex4, depth4
					}
				l5:
					goto l2
				l3:
					position, tokenIndex, depth = position3, tokenIndex3, depth3
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
		/* 1 Field <- <(('f' / 'F') ('i' / 'I') ('e' / 'E') ('l' / 'L') ('d' / 'D') s FieldName s '{' statements '}' S)> */
		func() bool {
			position6, tokenIndex6, depth6 := position, tokenIndex, depth
			{
				position7 := position
				depth++
				{
					position8, tokenIndex8, depth8 := position, tokenIndex, depth
					if buffer[position] != rune('f') {
						goto l9
					}
					position++
					goto l8
				l9:
					position, tokenIndex, depth = position8, tokenIndex8, depth8
					if buffer[position] != rune('F') {
						goto l6
					}
					position++
				}
			l8:
				{
					position10, tokenIndex10, depth10 := position, tokenIndex, depth
					if buffer[position] != rune('i') {
						goto l11
					}
					position++
					goto l10
				l11:
					position, tokenIndex, depth = position10, tokenIndex10, depth10
					if buffer[position] != rune('I') {
						goto l6
					}
					position++
				}
			l10:
				{
					position12, tokenIndex12, depth12 := position, tokenIndex, depth
					if buffer[position] != rune('e') {
						goto l13
					}
					position++
					goto l12
				l13:
					position, tokenIndex, depth = position12, tokenIndex12, depth12
					if buffer[position] != rune('E') {
						goto l6
					}
					position++
				}
			l12:
				{
					position14, tokenIndex14, depth14 := position, tokenIndex, depth
					if buffer[position] != rune('l') {
						goto l15
					}
					position++
					goto l14
				l15:
					position, tokenIndex, depth = position14, tokenIndex14, depth14
					if buffer[position] != rune('L') {
						goto l6
					}
					position++
				}
			l14:
				{
					position16, tokenIndex16, depth16 := position, tokenIndex, depth
					if buffer[position] != rune('d') {
						goto l17
					}
					position++
					goto l16
				l17:
					position, tokenIndex, depth = position16, tokenIndex16, depth16
					if buffer[position] != rune('D') {
						goto l6
					}
					position++
				}
			l16:
				if !_rules[rules]() {
					goto l6
				}
				if !_rules[ruleFieldName]() {
					goto l6
				}
				if !_rules[rules]() {
					goto l6
				}
				if buffer[position] != rune('{') {
					goto l6
				}
				position++
				if !_rules[rulestatements]() {
					goto l6
				}
				if buffer[position] != rune('}') {
					goto l6
				}
				position++
				if !_rules[ruleS]() {
					goto l6
				}
				depth--
				add(ruleField, position7)
			}
			return true
		l6:
			position, tokenIndex, depth = position6, tokenIndex6, depth6
			return false
		},
		/* 2 FieldName <- <(<([a-z] / [A-Z] / [0-9] / '-' / '_')+> Action0)> */
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
				add(ruleFieldName, position19)
			}
			return true
		l18:
			position, tokenIndex, depth = position18, tokenIndex18, depth18
			return false
		},
		/* 3 statements <- <((line EOL)* line?)> */
		func() bool {
			{
				position34 := position
				depth++
			l35:
				{
					position36, tokenIndex36, depth36 := position, tokenIndex, depth
					if !_rules[ruleline]() {
						goto l36
					}
					if !_rules[ruleEOL]() {
						goto l36
					}
					goto l35
				l36:
					position, tokenIndex, depth = position36, tokenIndex36, depth36
				}
				{
					position37, tokenIndex37, depth37 := position, tokenIndex, depth
					if !_rules[ruleline]() {
						goto l37
					}
					goto l38
				l37:
					position, tokenIndex, depth = position37, tokenIndex37, depth37
				}
			l38:
				depth--
				add(rulestatements, position34)
			}
			return true
		},
		/* 4 line <- <((Comment / Statement)? s)> */
		func() bool {
			position39, tokenIndex39, depth39 := position, tokenIndex, depth
			{
				position40 := position
				depth++
				{
					position41, tokenIndex41, depth41 := position, tokenIndex, depth
					{
						position43, tokenIndex43, depth43 := position, tokenIndex, depth
						if !_rules[ruleComment]() {
							goto l44
						}
						goto l43
					l44:
						position, tokenIndex, depth = position43, tokenIndex43, depth43
						if !_rules[ruleStatement]() {
							goto l41
						}
					}
				l43:
					goto l42
				l41:
					position, tokenIndex, depth = position41, tokenIndex41, depth41
				}
			l42:
				if !_rules[rules]() {
					goto l39
				}
				depth--
				add(ruleline, position40)
			}
			return true
		l39:
			position, tokenIndex, depth = position39, tokenIndex39, depth39
			return false
		},
		/* 5 Name <- <(<([a-z] / [A-Z] / [0-9] / '-' / '_')+> Action1)> */
		func() bool {
			position45, tokenIndex45, depth45 := position, tokenIndex, depth
			{
				position46 := position
				depth++
				{
					position47 := position
					depth++
					{
						position50, tokenIndex50, depth50 := position, tokenIndex, depth
						if c := buffer[position]; c < rune('a') || c > rune('z') {
							goto l51
						}
						position++
						goto l50
					l51:
						position, tokenIndex, depth = position50, tokenIndex50, depth50
						if c := buffer[position]; c < rune('A') || c > rune('Z') {
							goto l52
						}
						position++
						goto l50
					l52:
						position, tokenIndex, depth = position50, tokenIndex50, depth50
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l53
						}
						position++
						goto l50
					l53:
						position, tokenIndex, depth = position50, tokenIndex50, depth50
						if buffer[position] != rune('-') {
							goto l54
						}
						position++
						goto l50
					l54:
						position, tokenIndex, depth = position50, tokenIndex50, depth50
						if buffer[position] != rune('_') {
							goto l45
						}
						position++
					}
				l50:
				l48:
					{
						position49, tokenIndex49, depth49 := position, tokenIndex, depth
						{
							position55, tokenIndex55, depth55 := position, tokenIndex, depth
							if c := buffer[position]; c < rune('a') || c > rune('z') {
								goto l56
							}
							position++
							goto l55
						l56:
							position, tokenIndex, depth = position55, tokenIndex55, depth55
							if c := buffer[position]; c < rune('A') || c > rune('Z') {
								goto l57
							}
							position++
							goto l55
						l57:
							position, tokenIndex, depth = position55, tokenIndex55, depth55
							if c := buffer[position]; c < rune('0') || c > rune('9') {
								goto l58
							}
							position++
							goto l55
						l58:
							position, tokenIndex, depth = position55, tokenIndex55, depth55
							if buffer[position] != rune('-') {
								goto l59
							}
							position++
							goto l55
						l59:
							position, tokenIndex, depth = position55, tokenIndex55, depth55
							if buffer[position] != rune('_') {
								goto l49
							}
							position++
						}
					l55:
						goto l48
					l49:
						position, tokenIndex, depth = position49, tokenIndex49, depth49
					}
					depth--
					add(rulePegText, position47)
				}
				if !_rules[ruleAction1]() {
					goto l45
				}
				depth--
				add(ruleName, position46)
			}
			return true
		l45:
			position, tokenIndex, depth = position45, tokenIndex45, depth45
			return false
		},
		/* 6 Statement <- <(s Name s '=' s String Comment? Action2)> */
		func() bool {
			position60, tokenIndex60, depth60 := position, tokenIndex, depth
			{
				position61 := position
				depth++
				if !_rules[rules]() {
					goto l60
				}
				if !_rules[ruleName]() {
					goto l60
				}
				if !_rules[rules]() {
					goto l60
				}
				if buffer[position] != rune('=') {
					goto l60
				}
				position++
				if !_rules[rules]() {
					goto l60
				}
				if !_rules[ruleString]() {
					goto l60
				}
				{
					position62, tokenIndex62, depth62 := position, tokenIndex, depth
					if !_rules[ruleComment]() {
						goto l62
					}
					goto l63
				l62:
					position, tokenIndex, depth = position62, tokenIndex62, depth62
				}
			l63:
				if !_rules[ruleAction2]() {
					goto l60
				}
				depth--
				add(ruleStatement, position61)
			}
			return true
		l60:
			position, tokenIndex, depth = position60, tokenIndex60, depth60
			return false
		},
		/* 7 String <- <(DoubleQuotedString / SingleQuotedString / RawString)> */
		func() bool {
			position64, tokenIndex64, depth64 := position, tokenIndex, depth
			{
				position65 := position
				depth++
				{
					position66, tokenIndex66, depth66 := position, tokenIndex, depth
					if !_rules[ruleDoubleQuotedString]() {
						goto l67
					}
					goto l66
				l67:
					position, tokenIndex, depth = position66, tokenIndex66, depth66
					if !_rules[ruleSingleQuotedString]() {
						goto l68
					}
					goto l66
				l68:
					position, tokenIndex, depth = position66, tokenIndex66, depth66
					if !_rules[ruleRawString]() {
						goto l64
					}
				}
			l66:
				depth--
				add(ruleString, position65)
			}
			return true
		l64:
			position, tokenIndex, depth = position64, tokenIndex64, depth64
			return false
		},
		/* 8 SingleQuotedString <- <(<('\'' (('\\' '\'') / (!EOL !'\'' .))* '\'')> Action3)> */
		func() bool {
			position69, tokenIndex69, depth69 := position, tokenIndex, depth
			{
				position70 := position
				depth++
				{
					position71 := position
					depth++
					if buffer[position] != rune('\'') {
						goto l69
					}
					position++
				l72:
					{
						position73, tokenIndex73, depth73 := position, tokenIndex, depth
						{
							position74, tokenIndex74, depth74 := position, tokenIndex, depth
							if buffer[position] != rune('\\') {
								goto l75
							}
							position++
							if buffer[position] != rune('\'') {
								goto l75
							}
							position++
							goto l74
						l75:
							position, tokenIndex, depth = position74, tokenIndex74, depth74
							{
								position76, tokenIndex76, depth76 := position, tokenIndex, depth
								if !_rules[ruleEOL]() {
									goto l76
								}
								goto l73
							l76:
								position, tokenIndex, depth = position76, tokenIndex76, depth76
							}
							{
								position77, tokenIndex77, depth77 := position, tokenIndex, depth
								if buffer[position] != rune('\'') {
									goto l77
								}
								position++
								goto l73
							l77:
								position, tokenIndex, depth = position77, tokenIndex77, depth77
							}
							if !matchDot() {
								goto l73
							}
						}
					l74:
						goto l72
					l73:
						position, tokenIndex, depth = position73, tokenIndex73, depth73
					}
					if buffer[position] != rune('\'') {
						goto l69
					}
					position++
					depth--
					add(rulePegText, position71)
				}
				if !_rules[ruleAction3]() {
					goto l69
				}
				depth--
				add(ruleSingleQuotedString, position70)
			}
			return true
		l69:
			position, tokenIndex, depth = position69, tokenIndex69, depth69
			return false
		},
		/* 9 DoubleQuotedString <- <(<('"' (('\\' '"') / (!EOL !'"' .))* '"')> Action4)> */
		func() bool {
			position78, tokenIndex78, depth78 := position, tokenIndex, depth
			{
				position79 := position
				depth++
				{
					position80 := position
					depth++
					if buffer[position] != rune('"') {
						goto l78
					}
					position++
				l81:
					{
						position82, tokenIndex82, depth82 := position, tokenIndex, depth
						{
							position83, tokenIndex83, depth83 := position, tokenIndex, depth
							if buffer[position] != rune('\\') {
								goto l84
							}
							position++
							if buffer[position] != rune('"') {
								goto l84
							}
							position++
							goto l83
						l84:
							position, tokenIndex, depth = position83, tokenIndex83, depth83
							{
								position85, tokenIndex85, depth85 := position, tokenIndex, depth
								if !_rules[ruleEOL]() {
									goto l85
								}
								goto l82
							l85:
								position, tokenIndex, depth = position85, tokenIndex85, depth85
							}
							{
								position86, tokenIndex86, depth86 := position, tokenIndex, depth
								if buffer[position] != rune('"') {
									goto l86
								}
								position++
								goto l82
							l86:
								position, tokenIndex, depth = position86, tokenIndex86, depth86
							}
							if !matchDot() {
								goto l82
							}
						}
					l83:
						goto l81
					l82:
						position, tokenIndex, depth = position82, tokenIndex82, depth82
					}
					if buffer[position] != rune('"') {
						goto l78
					}
					position++
					depth--
					add(rulePegText, position80)
				}
				if !_rules[ruleAction4]() {
					goto l78
				}
				depth--
				add(ruleDoubleQuotedString, position79)
			}
			return true
		l78:
			position, tokenIndex, depth = position78, tokenIndex78, depth78
			return false
		},
		/* 10 RawString <- <(<('`' (!'`' .)* '`')> Action5)> */
		func() bool {
			position87, tokenIndex87, depth87 := position, tokenIndex, depth
			{
				position88 := position
				depth++
				{
					position89 := position
					depth++
					if buffer[position] != rune('`') {
						goto l87
					}
					position++
				l90:
					{
						position91, tokenIndex91, depth91 := position, tokenIndex, depth
						{
							position92, tokenIndex92, depth92 := position, tokenIndex, depth
							if buffer[position] != rune('`') {
								goto l92
							}
							position++
							goto l91
						l92:
							position, tokenIndex, depth = position92, tokenIndex92, depth92
						}
						if !matchDot() {
							goto l91
						}
						goto l90
					l91:
						position, tokenIndex, depth = position91, tokenIndex91, depth91
					}
					if buffer[position] != rune('`') {
						goto l87
					}
					position++
					depth--
					add(rulePegText, position89)
				}
				if !_rules[ruleAction5]() {
					goto l87
				}
				depth--
				add(ruleRawString, position88)
			}
			return true
		l87:
			position, tokenIndex, depth = position87, tokenIndex87, depth87
			return false
		},
		/* 11 Comment <- <(s '#' (!EOL .)*)> */
		func() bool {
			position93, tokenIndex93, depth93 := position, tokenIndex, depth
			{
				position94 := position
				depth++
				if !_rules[rules]() {
					goto l93
				}
				if buffer[position] != rune('#') {
					goto l93
				}
				position++
			l95:
				{
					position96, tokenIndex96, depth96 := position, tokenIndex, depth
					{
						position97, tokenIndex97, depth97 := position, tokenIndex, depth
						if !_rules[ruleEOL]() {
							goto l97
						}
						goto l96
					l97:
						position, tokenIndex, depth = position97, tokenIndex97, depth97
					}
					if !matchDot() {
						goto l96
					}
					goto l95
				l96:
					position, tokenIndex, depth = position96, tokenIndex96, depth96
				}
				depth--
				add(ruleComment, position94)
			}
			return true
		l93:
			position, tokenIndex, depth = position93, tokenIndex93, depth93
			return false
		},
		/* 12 EOF <- <!.> */
		func() bool {
			position98, tokenIndex98, depth98 := position, tokenIndex, depth
			{
				position99 := position
				depth++
				{
					position100, tokenIndex100, depth100 := position, tokenIndex, depth
					if !matchDot() {
						goto l100
					}
					goto l98
				l100:
					position, tokenIndex, depth = position100, tokenIndex100, depth100
				}
				depth--
				add(ruleEOF, position99)
			}
			return true
		l98:
			position, tokenIndex, depth = position98, tokenIndex98, depth98
			return false
		},
		/* 13 EOL <- <('\r' / '\n')> */
		func() bool {
			position101, tokenIndex101, depth101 := position, tokenIndex, depth
			{
				position102 := position
				depth++
				{
					position103, tokenIndex103, depth103 := position, tokenIndex, depth
					if buffer[position] != rune('\r') {
						goto l104
					}
					position++
					goto l103
				l104:
					position, tokenIndex, depth = position103, tokenIndex103, depth103
					if buffer[position] != rune('\n') {
						goto l101
					}
					position++
				}
			l103:
				depth--
				add(ruleEOL, position102)
			}
			return true
		l101:
			position, tokenIndex, depth = position101, tokenIndex101, depth101
			return false
		},
		/* 14 s <- <(' ' / '\t')*> */
		func() bool {
			{
				position106 := position
				depth++
			l107:
				{
					position108, tokenIndex108, depth108 := position, tokenIndex, depth
					{
						position109, tokenIndex109, depth109 := position, tokenIndex, depth
						if buffer[position] != rune(' ') {
							goto l110
						}
						position++
						goto l109
					l110:
						position, tokenIndex, depth = position109, tokenIndex109, depth109
						if buffer[position] != rune('\t') {
							goto l108
						}
						position++
					}
				l109:
					goto l107
				l108:
					position, tokenIndex, depth = position108, tokenIndex108, depth108
				}
				depth--
				add(rules, position106)
			}
			return true
		},
		/* 15 S <- <(s Comment? (Comment? EOL)*)> */
		func() bool {
			position111, tokenIndex111, depth111 := position, tokenIndex, depth
			{
				position112 := position
				depth++
				if !_rules[rules]() {
					goto l111
				}
				{
					position113, tokenIndex113, depth113 := position, tokenIndex, depth
					if !_rules[ruleComment]() {
						goto l113
					}
					goto l114
				l113:
					position, tokenIndex, depth = position113, tokenIndex113, depth113
				}
			l114:
			l115:
				{
					position116, tokenIndex116, depth116 := position, tokenIndex, depth
					{
						position117, tokenIndex117, depth117 := position, tokenIndex, depth
						if !_rules[ruleComment]() {
							goto l117
						}
						goto l118
					l117:
						position, tokenIndex, depth = position117, tokenIndex117, depth117
					}
				l118:
					if !_rules[ruleEOL]() {
						goto l116
					}
					goto l115
				l116:
					position, tokenIndex, depth = position116, tokenIndex116, depth116
				}
				depth--
				add(ruleS, position112)
			}
			return true
		l111:
			position, tokenIndex, depth = position111, tokenIndex111, depth111
			return false
		},
		nil,
		/* 18 Action0 <- <{ p.newField(buffer[begin:end]) }> */
		func() bool {
			{
				add(ruleAction0, position)
			}
			return true
		},
		/* 19 Action1 <- <{ p.name = buffer[begin:end] }> */
		func() bool {
			{
				add(ruleAction1, position)
			}
			return true
		},
		/* 20 Action2 <- <{ p.set(p.name, p.value) }> */
		func() bool {
			{
				add(ruleAction2, position)
			}
			return true
		},
		/* 21 Action3 <- <{ p.value = buffer[begin:end] }> */
		func() bool {
			{
				add(ruleAction3, position)
			}
			return true
		},
		/* 22 Action4 <- <{ p.value = buffer[begin:end] }> */
		func() bool {
			{
				add(ruleAction4, position)
			}
			return true
		},
		/* 23 Action5 <- <{ p.value = buffer[begin:end] }> */
		func() bool {
			{
				add(ruleAction5, position)
			}
			return true
		},
	}
	p.rules = _rules
}
