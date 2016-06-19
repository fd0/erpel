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
	ruleFields
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
	ruleSeparator
	ruleTemplates
	ruleTemplate
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
	ruleAction6

	rulePre
	ruleIn
	ruleSuf
)

var rul3s = [...]string{
	"Unknown",
	"start",
	"Fields",
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
	"Separator",
	"Templates",
	"Template",
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

type ruleParser struct {
	ruleState

	Buffer string
	buffer []rune
	rules  [29]func() bool
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
		case ruleAction6:
			p.addTemplate(buffer[begin:end])

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
		/* 0 start <- <(Fields (Separator Templates)? EOF)> */
		func() bool {
			position0, tokenIndex0, depth0 := position, tokenIndex, depth
			{
				position1 := position
				depth++
				if !_rules[ruleFields]() {
					goto l0
				}
				{
					position2, tokenIndex2, depth2 := position, tokenIndex, depth
					if !_rules[ruleSeparator]() {
						goto l2
					}
					if !_rules[ruleTemplates]() {
						goto l2
					}
					goto l3
				l2:
					position, tokenIndex, depth = position2, tokenIndex2, depth2
				}
			l3:
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
		/* 1 Fields <- <(S (Field EOL?)* Field?)> */
		func() bool {
			position4, tokenIndex4, depth4 := position, tokenIndex, depth
			{
				position5 := position
				depth++
				if !_rules[ruleS]() {
					goto l4
				}
			l6:
				{
					position7, tokenIndex7, depth7 := position, tokenIndex, depth
					if !_rules[ruleField]() {
						goto l7
					}
					{
						position8, tokenIndex8, depth8 := position, tokenIndex, depth
						if !_rules[ruleEOL]() {
							goto l8
						}
						goto l9
					l8:
						position, tokenIndex, depth = position8, tokenIndex8, depth8
					}
				l9:
					goto l6
				l7:
					position, tokenIndex, depth = position7, tokenIndex7, depth7
				}
				{
					position10, tokenIndex10, depth10 := position, tokenIndex, depth
					if !_rules[ruleField]() {
						goto l10
					}
					goto l11
				l10:
					position, tokenIndex, depth = position10, tokenIndex10, depth10
				}
			l11:
				depth--
				add(ruleFields, position5)
			}
			return true
		l4:
			position, tokenIndex, depth = position4, tokenIndex4, depth4
			return false
		},
		/* 2 Field <- <(('f' / 'F') ('i' / 'I') ('e' / 'E') ('l' / 'L') ('d' / 'D') s FieldName s '{' statements '}' S)> */
		func() bool {
			position12, tokenIndex12, depth12 := position, tokenIndex, depth
			{
				position13 := position
				depth++
				{
					position14, tokenIndex14, depth14 := position, tokenIndex, depth
					if buffer[position] != rune('f') {
						goto l15
					}
					position++
					goto l14
				l15:
					position, tokenIndex, depth = position14, tokenIndex14, depth14
					if buffer[position] != rune('F') {
						goto l12
					}
					position++
				}
			l14:
				{
					position16, tokenIndex16, depth16 := position, tokenIndex, depth
					if buffer[position] != rune('i') {
						goto l17
					}
					position++
					goto l16
				l17:
					position, tokenIndex, depth = position16, tokenIndex16, depth16
					if buffer[position] != rune('I') {
						goto l12
					}
					position++
				}
			l16:
				{
					position18, tokenIndex18, depth18 := position, tokenIndex, depth
					if buffer[position] != rune('e') {
						goto l19
					}
					position++
					goto l18
				l19:
					position, tokenIndex, depth = position18, tokenIndex18, depth18
					if buffer[position] != rune('E') {
						goto l12
					}
					position++
				}
			l18:
				{
					position20, tokenIndex20, depth20 := position, tokenIndex, depth
					if buffer[position] != rune('l') {
						goto l21
					}
					position++
					goto l20
				l21:
					position, tokenIndex, depth = position20, tokenIndex20, depth20
					if buffer[position] != rune('L') {
						goto l12
					}
					position++
				}
			l20:
				{
					position22, tokenIndex22, depth22 := position, tokenIndex, depth
					if buffer[position] != rune('d') {
						goto l23
					}
					position++
					goto l22
				l23:
					position, tokenIndex, depth = position22, tokenIndex22, depth22
					if buffer[position] != rune('D') {
						goto l12
					}
					position++
				}
			l22:
				if !_rules[rules]() {
					goto l12
				}
				if !_rules[ruleFieldName]() {
					goto l12
				}
				if !_rules[rules]() {
					goto l12
				}
				if buffer[position] != rune('{') {
					goto l12
				}
				position++
				if !_rules[rulestatements]() {
					goto l12
				}
				if buffer[position] != rune('}') {
					goto l12
				}
				position++
				if !_rules[ruleS]() {
					goto l12
				}
				depth--
				add(ruleField, position13)
			}
			return true
		l12:
			position, tokenIndex, depth = position12, tokenIndex12, depth12
			return false
		},
		/* 3 FieldName <- <(<([a-z] / [A-Z] / [0-9] / '-' / '_')+> Action0)> */
		func() bool {
			position24, tokenIndex24, depth24 := position, tokenIndex, depth
			{
				position25 := position
				depth++
				{
					position26 := position
					depth++
					{
						position29, tokenIndex29, depth29 := position, tokenIndex, depth
						if c := buffer[position]; c < rune('a') || c > rune('z') {
							goto l30
						}
						position++
						goto l29
					l30:
						position, tokenIndex, depth = position29, tokenIndex29, depth29
						if c := buffer[position]; c < rune('A') || c > rune('Z') {
							goto l31
						}
						position++
						goto l29
					l31:
						position, tokenIndex, depth = position29, tokenIndex29, depth29
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l32
						}
						position++
						goto l29
					l32:
						position, tokenIndex, depth = position29, tokenIndex29, depth29
						if buffer[position] != rune('-') {
							goto l33
						}
						position++
						goto l29
					l33:
						position, tokenIndex, depth = position29, tokenIndex29, depth29
						if buffer[position] != rune('_') {
							goto l24
						}
						position++
					}
				l29:
				l27:
					{
						position28, tokenIndex28, depth28 := position, tokenIndex, depth
						{
							position34, tokenIndex34, depth34 := position, tokenIndex, depth
							if c := buffer[position]; c < rune('a') || c > rune('z') {
								goto l35
							}
							position++
							goto l34
						l35:
							position, tokenIndex, depth = position34, tokenIndex34, depth34
							if c := buffer[position]; c < rune('A') || c > rune('Z') {
								goto l36
							}
							position++
							goto l34
						l36:
							position, tokenIndex, depth = position34, tokenIndex34, depth34
							if c := buffer[position]; c < rune('0') || c > rune('9') {
								goto l37
							}
							position++
							goto l34
						l37:
							position, tokenIndex, depth = position34, tokenIndex34, depth34
							if buffer[position] != rune('-') {
								goto l38
							}
							position++
							goto l34
						l38:
							position, tokenIndex, depth = position34, tokenIndex34, depth34
							if buffer[position] != rune('_') {
								goto l28
							}
							position++
						}
					l34:
						goto l27
					l28:
						position, tokenIndex, depth = position28, tokenIndex28, depth28
					}
					depth--
					add(rulePegText, position26)
				}
				if !_rules[ruleAction0]() {
					goto l24
				}
				depth--
				add(ruleFieldName, position25)
			}
			return true
		l24:
			position, tokenIndex, depth = position24, tokenIndex24, depth24
			return false
		},
		/* 4 statements <- <((line EOL)* line?)> */
		func() bool {
			{
				position40 := position
				depth++
			l41:
				{
					position42, tokenIndex42, depth42 := position, tokenIndex, depth
					if !_rules[ruleline]() {
						goto l42
					}
					if !_rules[ruleEOL]() {
						goto l42
					}
					goto l41
				l42:
					position, tokenIndex, depth = position42, tokenIndex42, depth42
				}
				{
					position43, tokenIndex43, depth43 := position, tokenIndex, depth
					if !_rules[ruleline]() {
						goto l43
					}
					goto l44
				l43:
					position, tokenIndex, depth = position43, tokenIndex43, depth43
				}
			l44:
				depth--
				add(rulestatements, position40)
			}
			return true
		},
		/* 5 line <- <((Comment / Statement)? s)> */
		func() bool {
			position45, tokenIndex45, depth45 := position, tokenIndex, depth
			{
				position46 := position
				depth++
				{
					position47, tokenIndex47, depth47 := position, tokenIndex, depth
					{
						position49, tokenIndex49, depth49 := position, tokenIndex, depth
						if !_rules[ruleComment]() {
							goto l50
						}
						goto l49
					l50:
						position, tokenIndex, depth = position49, tokenIndex49, depth49
						if !_rules[ruleStatement]() {
							goto l47
						}
					}
				l49:
					goto l48
				l47:
					position, tokenIndex, depth = position47, tokenIndex47, depth47
				}
			l48:
				if !_rules[rules]() {
					goto l45
				}
				depth--
				add(ruleline, position46)
			}
			return true
		l45:
			position, tokenIndex, depth = position45, tokenIndex45, depth45
			return false
		},
		/* 6 Name <- <(<([a-z] / [A-Z] / [0-9] / '-' / '_')+> Action1)> */
		func() bool {
			position51, tokenIndex51, depth51 := position, tokenIndex, depth
			{
				position52 := position
				depth++
				{
					position53 := position
					depth++
					{
						position56, tokenIndex56, depth56 := position, tokenIndex, depth
						if c := buffer[position]; c < rune('a') || c > rune('z') {
							goto l57
						}
						position++
						goto l56
					l57:
						position, tokenIndex, depth = position56, tokenIndex56, depth56
						if c := buffer[position]; c < rune('A') || c > rune('Z') {
							goto l58
						}
						position++
						goto l56
					l58:
						position, tokenIndex, depth = position56, tokenIndex56, depth56
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l59
						}
						position++
						goto l56
					l59:
						position, tokenIndex, depth = position56, tokenIndex56, depth56
						if buffer[position] != rune('-') {
							goto l60
						}
						position++
						goto l56
					l60:
						position, tokenIndex, depth = position56, tokenIndex56, depth56
						if buffer[position] != rune('_') {
							goto l51
						}
						position++
					}
				l56:
				l54:
					{
						position55, tokenIndex55, depth55 := position, tokenIndex, depth
						{
							position61, tokenIndex61, depth61 := position, tokenIndex, depth
							if c := buffer[position]; c < rune('a') || c > rune('z') {
								goto l62
							}
							position++
							goto l61
						l62:
							position, tokenIndex, depth = position61, tokenIndex61, depth61
							if c := buffer[position]; c < rune('A') || c > rune('Z') {
								goto l63
							}
							position++
							goto l61
						l63:
							position, tokenIndex, depth = position61, tokenIndex61, depth61
							if c := buffer[position]; c < rune('0') || c > rune('9') {
								goto l64
							}
							position++
							goto l61
						l64:
							position, tokenIndex, depth = position61, tokenIndex61, depth61
							if buffer[position] != rune('-') {
								goto l65
							}
							position++
							goto l61
						l65:
							position, tokenIndex, depth = position61, tokenIndex61, depth61
							if buffer[position] != rune('_') {
								goto l55
							}
							position++
						}
					l61:
						goto l54
					l55:
						position, tokenIndex, depth = position55, tokenIndex55, depth55
					}
					depth--
					add(rulePegText, position53)
				}
				if !_rules[ruleAction1]() {
					goto l51
				}
				depth--
				add(ruleName, position52)
			}
			return true
		l51:
			position, tokenIndex, depth = position51, tokenIndex51, depth51
			return false
		},
		/* 7 Statement <- <(s Name s '=' s String Comment? Action2)> */
		func() bool {
			position66, tokenIndex66, depth66 := position, tokenIndex, depth
			{
				position67 := position
				depth++
				if !_rules[rules]() {
					goto l66
				}
				if !_rules[ruleName]() {
					goto l66
				}
				if !_rules[rules]() {
					goto l66
				}
				if buffer[position] != rune('=') {
					goto l66
				}
				position++
				if !_rules[rules]() {
					goto l66
				}
				if !_rules[ruleString]() {
					goto l66
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
				if !_rules[ruleAction2]() {
					goto l66
				}
				depth--
				add(ruleStatement, position67)
			}
			return true
		l66:
			position, tokenIndex, depth = position66, tokenIndex66, depth66
			return false
		},
		/* 8 String <- <(DoubleQuotedString / SingleQuotedString / RawString)> */
		func() bool {
			position70, tokenIndex70, depth70 := position, tokenIndex, depth
			{
				position71 := position
				depth++
				{
					position72, tokenIndex72, depth72 := position, tokenIndex, depth
					if !_rules[ruleDoubleQuotedString]() {
						goto l73
					}
					goto l72
				l73:
					position, tokenIndex, depth = position72, tokenIndex72, depth72
					if !_rules[ruleSingleQuotedString]() {
						goto l74
					}
					goto l72
				l74:
					position, tokenIndex, depth = position72, tokenIndex72, depth72
					if !_rules[ruleRawString]() {
						goto l70
					}
				}
			l72:
				depth--
				add(ruleString, position71)
			}
			return true
		l70:
			position, tokenIndex, depth = position70, tokenIndex70, depth70
			return false
		},
		/* 9 SingleQuotedString <- <(<('\'' (('\\' '\'') / (!EOL !'\'' .))* '\'')> Action3)> */
		func() bool {
			position75, tokenIndex75, depth75 := position, tokenIndex, depth
			{
				position76 := position
				depth++
				{
					position77 := position
					depth++
					if buffer[position] != rune('\'') {
						goto l75
					}
					position++
				l78:
					{
						position79, tokenIndex79, depth79 := position, tokenIndex, depth
						{
							position80, tokenIndex80, depth80 := position, tokenIndex, depth
							if buffer[position] != rune('\\') {
								goto l81
							}
							position++
							if buffer[position] != rune('\'') {
								goto l81
							}
							position++
							goto l80
						l81:
							position, tokenIndex, depth = position80, tokenIndex80, depth80
							{
								position82, tokenIndex82, depth82 := position, tokenIndex, depth
								if !_rules[ruleEOL]() {
									goto l82
								}
								goto l79
							l82:
								position, tokenIndex, depth = position82, tokenIndex82, depth82
							}
							{
								position83, tokenIndex83, depth83 := position, tokenIndex, depth
								if buffer[position] != rune('\'') {
									goto l83
								}
								position++
								goto l79
							l83:
								position, tokenIndex, depth = position83, tokenIndex83, depth83
							}
							if !matchDot() {
								goto l79
							}
						}
					l80:
						goto l78
					l79:
						position, tokenIndex, depth = position79, tokenIndex79, depth79
					}
					if buffer[position] != rune('\'') {
						goto l75
					}
					position++
					depth--
					add(rulePegText, position77)
				}
				if !_rules[ruleAction3]() {
					goto l75
				}
				depth--
				add(ruleSingleQuotedString, position76)
			}
			return true
		l75:
			position, tokenIndex, depth = position75, tokenIndex75, depth75
			return false
		},
		/* 10 DoubleQuotedString <- <(<('"' (('\\' '"') / (!EOL !'"' .))* '"')> Action4)> */
		func() bool {
			position84, tokenIndex84, depth84 := position, tokenIndex, depth
			{
				position85 := position
				depth++
				{
					position86 := position
					depth++
					if buffer[position] != rune('"') {
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
							if buffer[position] != rune('"') {
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
								if buffer[position] != rune('"') {
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
					if buffer[position] != rune('"') {
						goto l84
					}
					position++
					depth--
					add(rulePegText, position86)
				}
				if !_rules[ruleAction4]() {
					goto l84
				}
				depth--
				add(ruleDoubleQuotedString, position85)
			}
			return true
		l84:
			position, tokenIndex, depth = position84, tokenIndex84, depth84
			return false
		},
		/* 11 RawString <- <(<('`' (!'`' .)* '`')> Action5)> */
		func() bool {
			position93, tokenIndex93, depth93 := position, tokenIndex, depth
			{
				position94 := position
				depth++
				{
					position95 := position
					depth++
					if buffer[position] != rune('`') {
						goto l93
					}
					position++
				l96:
					{
						position97, tokenIndex97, depth97 := position, tokenIndex, depth
						{
							position98, tokenIndex98, depth98 := position, tokenIndex, depth
							if buffer[position] != rune('`') {
								goto l98
							}
							position++
							goto l97
						l98:
							position, tokenIndex, depth = position98, tokenIndex98, depth98
						}
						if !matchDot() {
							goto l97
						}
						goto l96
					l97:
						position, tokenIndex, depth = position97, tokenIndex97, depth97
					}
					if buffer[position] != rune('`') {
						goto l93
					}
					position++
					depth--
					add(rulePegText, position95)
				}
				if !_rules[ruleAction5]() {
					goto l93
				}
				depth--
				add(ruleRawString, position94)
			}
			return true
		l93:
			position, tokenIndex, depth = position93, tokenIndex93, depth93
			return false
		},
		/* 12 Separator <- <(s ('-' '-' '-') '-'* s EOL)> */
		func() bool {
			position99, tokenIndex99, depth99 := position, tokenIndex, depth
			{
				position100 := position
				depth++
				if !_rules[rules]() {
					goto l99
				}
				if buffer[position] != rune('-') {
					goto l99
				}
				position++
				if buffer[position] != rune('-') {
					goto l99
				}
				position++
				if buffer[position] != rune('-') {
					goto l99
				}
				position++
			l101:
				{
					position102, tokenIndex102, depth102 := position, tokenIndex, depth
					if buffer[position] != rune('-') {
						goto l102
					}
					position++
					goto l101
				l102:
					position, tokenIndex, depth = position102, tokenIndex102, depth102
				}
				if !_rules[rules]() {
					goto l99
				}
				if !_rules[ruleEOL]() {
					goto l99
				}
				depth--
				add(ruleSeparator, position100)
			}
			return true
		l99:
			position, tokenIndex, depth = position99, tokenIndex99, depth99
			return false
		},
		/* 13 Templates <- <((Comment / Template) EOL)*> */
		func() bool {
			{
				position104 := position
				depth++
			l105:
				{
					position106, tokenIndex106, depth106 := position, tokenIndex, depth
					{
						position107, tokenIndex107, depth107 := position, tokenIndex, depth
						if !_rules[ruleComment]() {
							goto l108
						}
						goto l107
					l108:
						position, tokenIndex, depth = position107, tokenIndex107, depth107
						if !_rules[ruleTemplate]() {
							goto l106
						}
					}
				l107:
					if !_rules[ruleEOL]() {
						goto l106
					}
					goto l105
				l106:
					position, tokenIndex, depth = position106, tokenIndex106, depth106
				}
				depth--
				add(ruleTemplates, position104)
			}
			return true
		},
		/* 14 Template <- <(s <(!EOL .)*> Action6)> */
		func() bool {
			position109, tokenIndex109, depth109 := position, tokenIndex, depth
			{
				position110 := position
				depth++
				if !_rules[rules]() {
					goto l109
				}
				{
					position111 := position
					depth++
				l112:
					{
						position113, tokenIndex113, depth113 := position, tokenIndex, depth
						{
							position114, tokenIndex114, depth114 := position, tokenIndex, depth
							if !_rules[ruleEOL]() {
								goto l114
							}
							goto l113
						l114:
							position, tokenIndex, depth = position114, tokenIndex114, depth114
						}
						if !matchDot() {
							goto l113
						}
						goto l112
					l113:
						position, tokenIndex, depth = position113, tokenIndex113, depth113
					}
					depth--
					add(rulePegText, position111)
				}
				if !_rules[ruleAction6]() {
					goto l109
				}
				depth--
				add(ruleTemplate, position110)
			}
			return true
		l109:
			position, tokenIndex, depth = position109, tokenIndex109, depth109
			return false
		},
		/* 15 Comment <- <(s '#' (!EOL .)*)> */
		func() bool {
			position115, tokenIndex115, depth115 := position, tokenIndex, depth
			{
				position116 := position
				depth++
				if !_rules[rules]() {
					goto l115
				}
				if buffer[position] != rune('#') {
					goto l115
				}
				position++
			l117:
				{
					position118, tokenIndex118, depth118 := position, tokenIndex, depth
					{
						position119, tokenIndex119, depth119 := position, tokenIndex, depth
						if !_rules[ruleEOL]() {
							goto l119
						}
						goto l118
					l119:
						position, tokenIndex, depth = position119, tokenIndex119, depth119
					}
					if !matchDot() {
						goto l118
					}
					goto l117
				l118:
					position, tokenIndex, depth = position118, tokenIndex118, depth118
				}
				depth--
				add(ruleComment, position116)
			}
			return true
		l115:
			position, tokenIndex, depth = position115, tokenIndex115, depth115
			return false
		},
		/* 16 EOF <- <!.> */
		func() bool {
			position120, tokenIndex120, depth120 := position, tokenIndex, depth
			{
				position121 := position
				depth++
				{
					position122, tokenIndex122, depth122 := position, tokenIndex, depth
					if !matchDot() {
						goto l122
					}
					goto l120
				l122:
					position, tokenIndex, depth = position122, tokenIndex122, depth122
				}
				depth--
				add(ruleEOF, position121)
			}
			return true
		l120:
			position, tokenIndex, depth = position120, tokenIndex120, depth120
			return false
		},
		/* 17 EOL <- <('\r' / '\n')> */
		func() bool {
			position123, tokenIndex123, depth123 := position, tokenIndex, depth
			{
				position124 := position
				depth++
				{
					position125, tokenIndex125, depth125 := position, tokenIndex, depth
					if buffer[position] != rune('\r') {
						goto l126
					}
					position++
					goto l125
				l126:
					position, tokenIndex, depth = position125, tokenIndex125, depth125
					if buffer[position] != rune('\n') {
						goto l123
					}
					position++
				}
			l125:
				depth--
				add(ruleEOL, position124)
			}
			return true
		l123:
			position, tokenIndex, depth = position123, tokenIndex123, depth123
			return false
		},
		/* 18 s <- <(' ' / '\t')*> */
		func() bool {
			{
				position128 := position
				depth++
			l129:
				{
					position130, tokenIndex130, depth130 := position, tokenIndex, depth
					{
						position131, tokenIndex131, depth131 := position, tokenIndex, depth
						if buffer[position] != rune(' ') {
							goto l132
						}
						position++
						goto l131
					l132:
						position, tokenIndex, depth = position131, tokenIndex131, depth131
						if buffer[position] != rune('\t') {
							goto l130
						}
						position++
					}
				l131:
					goto l129
				l130:
					position, tokenIndex, depth = position130, tokenIndex130, depth130
				}
				depth--
				add(rules, position128)
			}
			return true
		},
		/* 19 S <- <(s Comment? (Comment? EOL)*)> */
		func() bool {
			position133, tokenIndex133, depth133 := position, tokenIndex, depth
			{
				position134 := position
				depth++
				if !_rules[rules]() {
					goto l133
				}
				{
					position135, tokenIndex135, depth135 := position, tokenIndex, depth
					if !_rules[ruleComment]() {
						goto l135
					}
					goto l136
				l135:
					position, tokenIndex, depth = position135, tokenIndex135, depth135
				}
			l136:
			l137:
				{
					position138, tokenIndex138, depth138 := position, tokenIndex, depth
					{
						position139, tokenIndex139, depth139 := position, tokenIndex, depth
						if !_rules[ruleComment]() {
							goto l139
						}
						goto l140
					l139:
						position, tokenIndex, depth = position139, tokenIndex139, depth139
					}
				l140:
					if !_rules[ruleEOL]() {
						goto l138
					}
					goto l137
				l138:
					position, tokenIndex, depth = position138, tokenIndex138, depth138
				}
				depth--
				add(ruleS, position134)
			}
			return true
		l133:
			position, tokenIndex, depth = position133, tokenIndex133, depth133
			return false
		},
		nil,
		/* 22 Action0 <- <{ p.newField(buffer[begin:end]) }> */
		func() bool {
			{
				add(ruleAction0, position)
			}
			return true
		},
		/* 23 Action1 <- <{ p.name = buffer[begin:end] }> */
		func() bool {
			{
				add(ruleAction1, position)
			}
			return true
		},
		/* 24 Action2 <- <{ p.set(p.name, p.value) }> */
		func() bool {
			{
				add(ruleAction2, position)
			}
			return true
		},
		/* 25 Action3 <- <{ p.value = buffer[begin:end] }> */
		func() bool {
			{
				add(ruleAction3, position)
			}
			return true
		},
		/* 26 Action4 <- <{ p.value = buffer[begin:end] }> */
		func() bool {
			{
				add(ruleAction4, position)
			}
			return true
		},
		/* 27 Action5 <- <{ p.value = buffer[begin:end] }> */
		func() bool {
			{
				add(ruleAction5, position)
			}
			return true
		},
		/* 28 Action6 <- <{ p.addTemplate(buffer[begin:end]) }> */
		func() bool {
			{
				add(ruleAction6, position)
			}
			return true
		},
	}
	p.rules = _rules
}
