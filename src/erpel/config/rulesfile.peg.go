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
	ruleFields
	ruleField
	ruleFieldName
	rulestatements
	ruleline
	ruleName
	ruleStatement
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
	ruleS
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
	"S",
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
			p.value = buffer[begin:end]
		case ruleAction7:
			p.addTemplate(buffer[begin:end])
		case ruleAction8:
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
		/* 0 start <- <(Fields (Separator Templates (Separator Samples)?)? EOF)> */
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
					{
						position4, tokenIndex4, depth4 := position, tokenIndex, depth
						if !_rules[ruleSeparator]() {
							goto l4
						}
						if !_rules[ruleSamples]() {
							goto l4
						}
						goto l5
					l4:
						position, tokenIndex, depth = position4, tokenIndex4, depth4
					}
				l5:
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
			position6, tokenIndex6, depth6 := position, tokenIndex, depth
			{
				position7 := position
				depth++
				if !_rules[ruleS]() {
					goto l6
				}
			l8:
				{
					position9, tokenIndex9, depth9 := position, tokenIndex, depth
					if !_rules[ruleField]() {
						goto l9
					}
					{
						position10, tokenIndex10, depth10 := position, tokenIndex, depth
						if !_rules[ruleEOL]() {
							goto l10
						}
						goto l11
					l10:
						position, tokenIndex, depth = position10, tokenIndex10, depth10
					}
				l11:
					goto l8
				l9:
					position, tokenIndex, depth = position9, tokenIndex9, depth9
				}
				{
					position12, tokenIndex12, depth12 := position, tokenIndex, depth
					if !_rules[ruleField]() {
						goto l12
					}
					goto l13
				l12:
					position, tokenIndex, depth = position12, tokenIndex12, depth12
				}
			l13:
				depth--
				add(ruleFields, position7)
			}
			return true
		l6:
			position, tokenIndex, depth = position6, tokenIndex6, depth6
			return false
		},
		/* 2 Field <- <(('f' / 'F') ('i' / 'I') ('e' / 'E') ('l' / 'L') ('d' / 'D') s FieldName s '{' statements '}' S)> */
		func() bool {
			position14, tokenIndex14, depth14 := position, tokenIndex, depth
			{
				position15 := position
				depth++
				{
					position16, tokenIndex16, depth16 := position, tokenIndex, depth
					if buffer[position] != rune('f') {
						goto l17
					}
					position++
					goto l16
				l17:
					position, tokenIndex, depth = position16, tokenIndex16, depth16
					if buffer[position] != rune('F') {
						goto l14
					}
					position++
				}
			l16:
				{
					position18, tokenIndex18, depth18 := position, tokenIndex, depth
					if buffer[position] != rune('i') {
						goto l19
					}
					position++
					goto l18
				l19:
					position, tokenIndex, depth = position18, tokenIndex18, depth18
					if buffer[position] != rune('I') {
						goto l14
					}
					position++
				}
			l18:
				{
					position20, tokenIndex20, depth20 := position, tokenIndex, depth
					if buffer[position] != rune('e') {
						goto l21
					}
					position++
					goto l20
				l21:
					position, tokenIndex, depth = position20, tokenIndex20, depth20
					if buffer[position] != rune('E') {
						goto l14
					}
					position++
				}
			l20:
				{
					position22, tokenIndex22, depth22 := position, tokenIndex, depth
					if buffer[position] != rune('l') {
						goto l23
					}
					position++
					goto l22
				l23:
					position, tokenIndex, depth = position22, tokenIndex22, depth22
					if buffer[position] != rune('L') {
						goto l14
					}
					position++
				}
			l22:
				{
					position24, tokenIndex24, depth24 := position, tokenIndex, depth
					if buffer[position] != rune('d') {
						goto l25
					}
					position++
					goto l24
				l25:
					position, tokenIndex, depth = position24, tokenIndex24, depth24
					if buffer[position] != rune('D') {
						goto l14
					}
					position++
				}
			l24:
				if !_rules[rules]() {
					goto l14
				}
				if !_rules[ruleFieldName]() {
					goto l14
				}
				if !_rules[rules]() {
					goto l14
				}
				if buffer[position] != rune('{') {
					goto l14
				}
				position++
				if !_rules[rulestatements]() {
					goto l14
				}
				if buffer[position] != rune('}') {
					goto l14
				}
				position++
				if !_rules[ruleS]() {
					goto l14
				}
				depth--
				add(ruleField, position15)
			}
			return true
		l14:
			position, tokenIndex, depth = position14, tokenIndex14, depth14
			return false
		},
		/* 3 FieldName <- <(<([a-z] / [A-Z] / [0-9] / '-' / '_')+> Action0)> */
		func() bool {
			position26, tokenIndex26, depth26 := position, tokenIndex, depth
			{
				position27 := position
				depth++
				{
					position28 := position
					depth++
					{
						position31, tokenIndex31, depth31 := position, tokenIndex, depth
						if c := buffer[position]; c < rune('a') || c > rune('z') {
							goto l32
						}
						position++
						goto l31
					l32:
						position, tokenIndex, depth = position31, tokenIndex31, depth31
						if c := buffer[position]; c < rune('A') || c > rune('Z') {
							goto l33
						}
						position++
						goto l31
					l33:
						position, tokenIndex, depth = position31, tokenIndex31, depth31
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l34
						}
						position++
						goto l31
					l34:
						position, tokenIndex, depth = position31, tokenIndex31, depth31
						if buffer[position] != rune('-') {
							goto l35
						}
						position++
						goto l31
					l35:
						position, tokenIndex, depth = position31, tokenIndex31, depth31
						if buffer[position] != rune('_') {
							goto l26
						}
						position++
					}
				l31:
				l29:
					{
						position30, tokenIndex30, depth30 := position, tokenIndex, depth
						{
							position36, tokenIndex36, depth36 := position, tokenIndex, depth
							if c := buffer[position]; c < rune('a') || c > rune('z') {
								goto l37
							}
							position++
							goto l36
						l37:
							position, tokenIndex, depth = position36, tokenIndex36, depth36
							if c := buffer[position]; c < rune('A') || c > rune('Z') {
								goto l38
							}
							position++
							goto l36
						l38:
							position, tokenIndex, depth = position36, tokenIndex36, depth36
							if c := buffer[position]; c < rune('0') || c > rune('9') {
								goto l39
							}
							position++
							goto l36
						l39:
							position, tokenIndex, depth = position36, tokenIndex36, depth36
							if buffer[position] != rune('-') {
								goto l40
							}
							position++
							goto l36
						l40:
							position, tokenIndex, depth = position36, tokenIndex36, depth36
							if buffer[position] != rune('_') {
								goto l30
							}
							position++
						}
					l36:
						goto l29
					l30:
						position, tokenIndex, depth = position30, tokenIndex30, depth30
					}
					depth--
					add(rulePegText, position28)
				}
				if !_rules[ruleAction0]() {
					goto l26
				}
				depth--
				add(ruleFieldName, position27)
			}
			return true
		l26:
			position, tokenIndex, depth = position26, tokenIndex26, depth26
			return false
		},
		/* 4 statements <- <((line EOL)* line?)> */
		func() bool {
			{
				position42 := position
				depth++
			l43:
				{
					position44, tokenIndex44, depth44 := position, tokenIndex, depth
					if !_rules[ruleline]() {
						goto l44
					}
					if !_rules[ruleEOL]() {
						goto l44
					}
					goto l43
				l44:
					position, tokenIndex, depth = position44, tokenIndex44, depth44
				}
				{
					position45, tokenIndex45, depth45 := position, tokenIndex, depth
					if !_rules[ruleline]() {
						goto l45
					}
					goto l46
				l45:
					position, tokenIndex, depth = position45, tokenIndex45, depth45
				}
			l46:
				depth--
				add(rulestatements, position42)
			}
			return true
		},
		/* 5 line <- <((Comment / Statement)? s)> */
		func() bool {
			position47, tokenIndex47, depth47 := position, tokenIndex, depth
			{
				position48 := position
				depth++
				{
					position49, tokenIndex49, depth49 := position, tokenIndex, depth
					{
						position51, tokenIndex51, depth51 := position, tokenIndex, depth
						if !_rules[ruleComment]() {
							goto l52
						}
						goto l51
					l52:
						position, tokenIndex, depth = position51, tokenIndex51, depth51
						if !_rules[ruleStatement]() {
							goto l49
						}
					}
				l51:
					goto l50
				l49:
					position, tokenIndex, depth = position49, tokenIndex49, depth49
				}
			l50:
				if !_rules[rules]() {
					goto l47
				}
				depth--
				add(ruleline, position48)
			}
			return true
		l47:
			position, tokenIndex, depth = position47, tokenIndex47, depth47
			return false
		},
		/* 6 Name <- <(<([a-z] / [A-Z] / [0-9] / '-' / '_')+> Action1)> */
		func() bool {
			position53, tokenIndex53, depth53 := position, tokenIndex, depth
			{
				position54 := position
				depth++
				{
					position55 := position
					depth++
					{
						position58, tokenIndex58, depth58 := position, tokenIndex, depth
						if c := buffer[position]; c < rune('a') || c > rune('z') {
							goto l59
						}
						position++
						goto l58
					l59:
						position, tokenIndex, depth = position58, tokenIndex58, depth58
						if c := buffer[position]; c < rune('A') || c > rune('Z') {
							goto l60
						}
						position++
						goto l58
					l60:
						position, tokenIndex, depth = position58, tokenIndex58, depth58
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l61
						}
						position++
						goto l58
					l61:
						position, tokenIndex, depth = position58, tokenIndex58, depth58
						if buffer[position] != rune('-') {
							goto l62
						}
						position++
						goto l58
					l62:
						position, tokenIndex, depth = position58, tokenIndex58, depth58
						if buffer[position] != rune('_') {
							goto l53
						}
						position++
					}
				l58:
				l56:
					{
						position57, tokenIndex57, depth57 := position, tokenIndex, depth
						{
							position63, tokenIndex63, depth63 := position, tokenIndex, depth
							if c := buffer[position]; c < rune('a') || c > rune('z') {
								goto l64
							}
							position++
							goto l63
						l64:
							position, tokenIndex, depth = position63, tokenIndex63, depth63
							if c := buffer[position]; c < rune('A') || c > rune('Z') {
								goto l65
							}
							position++
							goto l63
						l65:
							position, tokenIndex, depth = position63, tokenIndex63, depth63
							if c := buffer[position]; c < rune('0') || c > rune('9') {
								goto l66
							}
							position++
							goto l63
						l66:
							position, tokenIndex, depth = position63, tokenIndex63, depth63
							if buffer[position] != rune('-') {
								goto l67
							}
							position++
							goto l63
						l67:
							position, tokenIndex, depth = position63, tokenIndex63, depth63
							if buffer[position] != rune('_') {
								goto l57
							}
							position++
						}
					l63:
						goto l56
					l57:
						position, tokenIndex, depth = position57, tokenIndex57, depth57
					}
					depth--
					add(rulePegText, position55)
				}
				if !_rules[ruleAction1]() {
					goto l53
				}
				depth--
				add(ruleName, position54)
			}
			return true
		l53:
			position, tokenIndex, depth = position53, tokenIndex53, depth53
			return false
		},
		/* 7 Statement <- <(s Name s '=' s Value Comment? Action2)> */
		func() bool {
			position68, tokenIndex68, depth68 := position, tokenIndex, depth
			{
				position69 := position
				depth++
				if !_rules[rules]() {
					goto l68
				}
				if !_rules[ruleName]() {
					goto l68
				}
				if !_rules[rules]() {
					goto l68
				}
				if buffer[position] != rune('=') {
					goto l68
				}
				position++
				if !_rules[rules]() {
					goto l68
				}
				if !_rules[ruleValue]() {
					goto l68
				}
				{
					position70, tokenIndex70, depth70 := position, tokenIndex, depth
					if !_rules[ruleComment]() {
						goto l70
					}
					goto l71
				l70:
					position, tokenIndex, depth = position70, tokenIndex70, depth70
				}
			l71:
				if !_rules[ruleAction2]() {
					goto l68
				}
				depth--
				add(ruleStatement, position69)
			}
			return true
		l68:
			position, tokenIndex, depth = position68, tokenIndex68, depth68
			return false
		},
		/* 8 Value <- <(List / String)> */
		func() bool {
			position72, tokenIndex72, depth72 := position, tokenIndex, depth
			{
				position73 := position
				depth++
				{
					position74, tokenIndex74, depth74 := position, tokenIndex, depth
					if !_rules[ruleList]() {
						goto l75
					}
					goto l74
				l75:
					position, tokenIndex, depth = position74, tokenIndex74, depth74
					if !_rules[ruleString]() {
						goto l72
					}
				}
			l74:
				depth--
				add(ruleValue, position73)
			}
			return true
		l72:
			position, tokenIndex, depth = position72, tokenIndex72, depth72
			return false
		},
		/* 9 String <- <(DoubleQuotedString / SingleQuotedString / RawString)> */
		func() bool {
			position76, tokenIndex76, depth76 := position, tokenIndex, depth
			{
				position77 := position
				depth++
				{
					position78, tokenIndex78, depth78 := position, tokenIndex, depth
					if !_rules[ruleDoubleQuotedString]() {
						goto l79
					}
					goto l78
				l79:
					position, tokenIndex, depth = position78, tokenIndex78, depth78
					if !_rules[ruleSingleQuotedString]() {
						goto l80
					}
					goto l78
				l80:
					position, tokenIndex, depth = position78, tokenIndex78, depth78
					if !_rules[ruleRawString]() {
						goto l76
					}
				}
			l78:
				depth--
				add(ruleString, position77)
			}
			return true
		l76:
			position, tokenIndex, depth = position76, tokenIndex76, depth76
			return false
		},
		/* 10 List <- <(<('[' s (s String s ',' s)* s String s ']')> Action3)> */
		func() bool {
			position81, tokenIndex81, depth81 := position, tokenIndex, depth
			{
				position82 := position
				depth++
				{
					position83 := position
					depth++
					if buffer[position] != rune('[') {
						goto l81
					}
					position++
					if !_rules[rules]() {
						goto l81
					}
				l84:
					{
						position85, tokenIndex85, depth85 := position, tokenIndex, depth
						if !_rules[rules]() {
							goto l85
						}
						if !_rules[ruleString]() {
							goto l85
						}
						if !_rules[rules]() {
							goto l85
						}
						if buffer[position] != rune(',') {
							goto l85
						}
						position++
						if !_rules[rules]() {
							goto l85
						}
						goto l84
					l85:
						position, tokenIndex, depth = position85, tokenIndex85, depth85
					}
					if !_rules[rules]() {
						goto l81
					}
					if !_rules[ruleString]() {
						goto l81
					}
					if !_rules[rules]() {
						goto l81
					}
					if buffer[position] != rune(']') {
						goto l81
					}
					position++
					depth--
					add(rulePegText, position83)
				}
				if !_rules[ruleAction3]() {
					goto l81
				}
				depth--
				add(ruleList, position82)
			}
			return true
		l81:
			position, tokenIndex, depth = position81, tokenIndex81, depth81
			return false
		},
		/* 11 SingleQuotedString <- <(<('\'' (('\\' '\'') / (!EOL !'\'' .))* '\'')> Action4)> */
		func() bool {
			position86, tokenIndex86, depth86 := position, tokenIndex, depth
			{
				position87 := position
				depth++
				{
					position88 := position
					depth++
					if buffer[position] != rune('\'') {
						goto l86
					}
					position++
				l89:
					{
						position90, tokenIndex90, depth90 := position, tokenIndex, depth
						{
							position91, tokenIndex91, depth91 := position, tokenIndex, depth
							if buffer[position] != rune('\\') {
								goto l92
							}
							position++
							if buffer[position] != rune('\'') {
								goto l92
							}
							position++
							goto l91
						l92:
							position, tokenIndex, depth = position91, tokenIndex91, depth91
							{
								position93, tokenIndex93, depth93 := position, tokenIndex, depth
								if !_rules[ruleEOL]() {
									goto l93
								}
								goto l90
							l93:
								position, tokenIndex, depth = position93, tokenIndex93, depth93
							}
							{
								position94, tokenIndex94, depth94 := position, tokenIndex, depth
								if buffer[position] != rune('\'') {
									goto l94
								}
								position++
								goto l90
							l94:
								position, tokenIndex, depth = position94, tokenIndex94, depth94
							}
							if !matchDot() {
								goto l90
							}
						}
					l91:
						goto l89
					l90:
						position, tokenIndex, depth = position90, tokenIndex90, depth90
					}
					if buffer[position] != rune('\'') {
						goto l86
					}
					position++
					depth--
					add(rulePegText, position88)
				}
				if !_rules[ruleAction4]() {
					goto l86
				}
				depth--
				add(ruleSingleQuotedString, position87)
			}
			return true
		l86:
			position, tokenIndex, depth = position86, tokenIndex86, depth86
			return false
		},
		/* 12 DoubleQuotedString <- <(<('"' (('\\' '"') / (!EOL !'"' .))* '"')> Action5)> */
		func() bool {
			position95, tokenIndex95, depth95 := position, tokenIndex, depth
			{
				position96 := position
				depth++
				{
					position97 := position
					depth++
					if buffer[position] != rune('"') {
						goto l95
					}
					position++
				l98:
					{
						position99, tokenIndex99, depth99 := position, tokenIndex, depth
						{
							position100, tokenIndex100, depth100 := position, tokenIndex, depth
							if buffer[position] != rune('\\') {
								goto l101
							}
							position++
							if buffer[position] != rune('"') {
								goto l101
							}
							position++
							goto l100
						l101:
							position, tokenIndex, depth = position100, tokenIndex100, depth100
							{
								position102, tokenIndex102, depth102 := position, tokenIndex, depth
								if !_rules[ruleEOL]() {
									goto l102
								}
								goto l99
							l102:
								position, tokenIndex, depth = position102, tokenIndex102, depth102
							}
							{
								position103, tokenIndex103, depth103 := position, tokenIndex, depth
								if buffer[position] != rune('"') {
									goto l103
								}
								position++
								goto l99
							l103:
								position, tokenIndex, depth = position103, tokenIndex103, depth103
							}
							if !matchDot() {
								goto l99
							}
						}
					l100:
						goto l98
					l99:
						position, tokenIndex, depth = position99, tokenIndex99, depth99
					}
					if buffer[position] != rune('"') {
						goto l95
					}
					position++
					depth--
					add(rulePegText, position97)
				}
				if !_rules[ruleAction5]() {
					goto l95
				}
				depth--
				add(ruleDoubleQuotedString, position96)
			}
			return true
		l95:
			position, tokenIndex, depth = position95, tokenIndex95, depth95
			return false
		},
		/* 13 RawString <- <(<('`' (!'`' .)* '`')> Action6)> */
		func() bool {
			position104, tokenIndex104, depth104 := position, tokenIndex, depth
			{
				position105 := position
				depth++
				{
					position106 := position
					depth++
					if buffer[position] != rune('`') {
						goto l104
					}
					position++
				l107:
					{
						position108, tokenIndex108, depth108 := position, tokenIndex, depth
						{
							position109, tokenIndex109, depth109 := position, tokenIndex, depth
							if buffer[position] != rune('`') {
								goto l109
							}
							position++
							goto l108
						l109:
							position, tokenIndex, depth = position109, tokenIndex109, depth109
						}
						if !matchDot() {
							goto l108
						}
						goto l107
					l108:
						position, tokenIndex, depth = position108, tokenIndex108, depth108
					}
					if buffer[position] != rune('`') {
						goto l104
					}
					position++
					depth--
					add(rulePegText, position106)
				}
				if !_rules[ruleAction6]() {
					goto l104
				}
				depth--
				add(ruleRawString, position105)
			}
			return true
		l104:
			position, tokenIndex, depth = position104, tokenIndex104, depth104
			return false
		},
		/* 14 Separator <- <(s ('-' '-' '-') '-'* s EOL)> */
		func() bool {
			position110, tokenIndex110, depth110 := position, tokenIndex, depth
			{
				position111 := position
				depth++
				if !_rules[rules]() {
					goto l110
				}
				if buffer[position] != rune('-') {
					goto l110
				}
				position++
				if buffer[position] != rune('-') {
					goto l110
				}
				position++
				if buffer[position] != rune('-') {
					goto l110
				}
				position++
			l112:
				{
					position113, tokenIndex113, depth113 := position, tokenIndex, depth
					if buffer[position] != rune('-') {
						goto l113
					}
					position++
					goto l112
				l113:
					position, tokenIndex, depth = position113, tokenIndex113, depth113
				}
				if !_rules[rules]() {
					goto l110
				}
				if !_rules[ruleEOL]() {
					goto l110
				}
				depth--
				add(ruleSeparator, position111)
			}
			return true
		l110:
			position, tokenIndex, depth = position110, tokenIndex110, depth110
			return false
		},
		/* 15 Templates <- <(!Separator (Comment / Template) EOL)*> */
		func() bool {
			{
				position115 := position
				depth++
			l116:
				{
					position117, tokenIndex117, depth117 := position, tokenIndex, depth
					{
						position118, tokenIndex118, depth118 := position, tokenIndex, depth
						if !_rules[ruleSeparator]() {
							goto l118
						}
						goto l117
					l118:
						position, tokenIndex, depth = position118, tokenIndex118, depth118
					}
					{
						position119, tokenIndex119, depth119 := position, tokenIndex, depth
						if !_rules[ruleComment]() {
							goto l120
						}
						goto l119
					l120:
						position, tokenIndex, depth = position119, tokenIndex119, depth119
						if !_rules[ruleTemplate]() {
							goto l117
						}
					}
				l119:
					if !_rules[ruleEOL]() {
						goto l117
					}
					goto l116
				l117:
					position, tokenIndex, depth = position117, tokenIndex117, depth117
				}
				depth--
				add(ruleTemplates, position115)
			}
			return true
		},
		/* 16 Template <- <(s <(!EOL .)*> Action7)> */
		func() bool {
			position121, tokenIndex121, depth121 := position, tokenIndex, depth
			{
				position122 := position
				depth++
				if !_rules[rules]() {
					goto l121
				}
				{
					position123 := position
					depth++
				l124:
					{
						position125, tokenIndex125, depth125 := position, tokenIndex, depth
						{
							position126, tokenIndex126, depth126 := position, tokenIndex, depth
							if !_rules[ruleEOL]() {
								goto l126
							}
							goto l125
						l126:
							position, tokenIndex, depth = position126, tokenIndex126, depth126
						}
						if !matchDot() {
							goto l125
						}
						goto l124
					l125:
						position, tokenIndex, depth = position125, tokenIndex125, depth125
					}
					depth--
					add(rulePegText, position123)
				}
				if !_rules[ruleAction7]() {
					goto l121
				}
				depth--
				add(ruleTemplate, position122)
			}
			return true
		l121:
			position, tokenIndex, depth = position121, tokenIndex121, depth121
			return false
		},
		/* 17 Samples <- <((Comment / Sample) EOL)*> */
		func() bool {
			{
				position128 := position
				depth++
			l129:
				{
					position130, tokenIndex130, depth130 := position, tokenIndex, depth
					{
						position131, tokenIndex131, depth131 := position, tokenIndex, depth
						if !_rules[ruleComment]() {
							goto l132
						}
						goto l131
					l132:
						position, tokenIndex, depth = position131, tokenIndex131, depth131
						if !_rules[ruleSample]() {
							goto l130
						}
					}
				l131:
					if !_rules[ruleEOL]() {
						goto l130
					}
					goto l129
				l130:
					position, tokenIndex, depth = position130, tokenIndex130, depth130
				}
				depth--
				add(ruleSamples, position128)
			}
			return true
		},
		/* 18 Sample <- <(s <(!EOL .)*> Action8)> */
		func() bool {
			position133, tokenIndex133, depth133 := position, tokenIndex, depth
			{
				position134 := position
				depth++
				if !_rules[rules]() {
					goto l133
				}
				{
					position135 := position
					depth++
				l136:
					{
						position137, tokenIndex137, depth137 := position, tokenIndex, depth
						{
							position138, tokenIndex138, depth138 := position, tokenIndex, depth
							if !_rules[ruleEOL]() {
								goto l138
							}
							goto l137
						l138:
							position, tokenIndex, depth = position138, tokenIndex138, depth138
						}
						if !matchDot() {
							goto l137
						}
						goto l136
					l137:
						position, tokenIndex, depth = position137, tokenIndex137, depth137
					}
					depth--
					add(rulePegText, position135)
				}
				if !_rules[ruleAction8]() {
					goto l133
				}
				depth--
				add(ruleSample, position134)
			}
			return true
		l133:
			position, tokenIndex, depth = position133, tokenIndex133, depth133
			return false
		},
		/* 19 Comment <- <(s '#' (!EOL .)*)> */
		func() bool {
			position139, tokenIndex139, depth139 := position, tokenIndex, depth
			{
				position140 := position
				depth++
				if !_rules[rules]() {
					goto l139
				}
				if buffer[position] != rune('#') {
					goto l139
				}
				position++
			l141:
				{
					position142, tokenIndex142, depth142 := position, tokenIndex, depth
					{
						position143, tokenIndex143, depth143 := position, tokenIndex, depth
						if !_rules[ruleEOL]() {
							goto l143
						}
						goto l142
					l143:
						position, tokenIndex, depth = position143, tokenIndex143, depth143
					}
					if !matchDot() {
						goto l142
					}
					goto l141
				l142:
					position, tokenIndex, depth = position142, tokenIndex142, depth142
				}
				depth--
				add(ruleComment, position140)
			}
			return true
		l139:
			position, tokenIndex, depth = position139, tokenIndex139, depth139
			return false
		},
		/* 20 EOF <- <!.> */
		func() bool {
			position144, tokenIndex144, depth144 := position, tokenIndex, depth
			{
				position145 := position
				depth++
				{
					position146, tokenIndex146, depth146 := position, tokenIndex, depth
					if !matchDot() {
						goto l146
					}
					goto l144
				l146:
					position, tokenIndex, depth = position146, tokenIndex146, depth146
				}
				depth--
				add(ruleEOF, position145)
			}
			return true
		l144:
			position, tokenIndex, depth = position144, tokenIndex144, depth144
			return false
		},
		/* 21 EOL <- <('\r' / '\n')> */
		func() bool {
			position147, tokenIndex147, depth147 := position, tokenIndex, depth
			{
				position148 := position
				depth++
				{
					position149, tokenIndex149, depth149 := position, tokenIndex, depth
					if buffer[position] != rune('\r') {
						goto l150
					}
					position++
					goto l149
				l150:
					position, tokenIndex, depth = position149, tokenIndex149, depth149
					if buffer[position] != rune('\n') {
						goto l147
					}
					position++
				}
			l149:
				depth--
				add(ruleEOL, position148)
			}
			return true
		l147:
			position, tokenIndex, depth = position147, tokenIndex147, depth147
			return false
		},
		/* 22 s <- <(' ' / '\t')*> */
		func() bool {
			{
				position152 := position
				depth++
			l153:
				{
					position154, tokenIndex154, depth154 := position, tokenIndex, depth
					{
						position155, tokenIndex155, depth155 := position, tokenIndex, depth
						if buffer[position] != rune(' ') {
							goto l156
						}
						position++
						goto l155
					l156:
						position, tokenIndex, depth = position155, tokenIndex155, depth155
						if buffer[position] != rune('\t') {
							goto l154
						}
						position++
					}
				l155:
					goto l153
				l154:
					position, tokenIndex, depth = position154, tokenIndex154, depth154
				}
				depth--
				add(rules, position152)
			}
			return true
		},
		/* 23 S <- <(s Comment? (Comment? EOL)*)> */
		func() bool {
			position157, tokenIndex157, depth157 := position, tokenIndex, depth
			{
				position158 := position
				depth++
				if !_rules[rules]() {
					goto l157
				}
				{
					position159, tokenIndex159, depth159 := position, tokenIndex, depth
					if !_rules[ruleComment]() {
						goto l159
					}
					goto l160
				l159:
					position, tokenIndex, depth = position159, tokenIndex159, depth159
				}
			l160:
			l161:
				{
					position162, tokenIndex162, depth162 := position, tokenIndex, depth
					{
						position163, tokenIndex163, depth163 := position, tokenIndex, depth
						if !_rules[ruleComment]() {
							goto l163
						}
						goto l164
					l163:
						position, tokenIndex, depth = position163, tokenIndex163, depth163
					}
				l164:
					if !_rules[ruleEOL]() {
						goto l162
					}
					goto l161
				l162:
					position, tokenIndex, depth = position162, tokenIndex162, depth162
				}
				depth--
				add(ruleS, position158)
			}
			return true
		l157:
			position, tokenIndex, depth = position157, tokenIndex157, depth157
			return false
		},
		nil,
		/* 26 Action0 <- <{ p.newField(buffer[begin:end]) }> */
		func() bool {
			{
				add(ruleAction0, position)
			}
			return true
		},
		/* 27 Action1 <- <{ p.name = buffer[begin:end] }> */
		func() bool {
			{
				add(ruleAction1, position)
			}
			return true
		},
		/* 28 Action2 <- <{ p.set(p.name, p.value) }> */
		func() bool {
			{
				add(ruleAction2, position)
			}
			return true
		},
		/* 29 Action3 <- <{ p.value = buffer[begin:end] }> */
		func() bool {
			{
				add(ruleAction3, position)
			}
			return true
		},
		/* 30 Action4 <- <{ p.value = buffer[begin:end] }> */
		func() bool {
			{
				add(ruleAction4, position)
			}
			return true
		},
		/* 31 Action5 <- <{ p.value = buffer[begin:end] }> */
		func() bool {
			{
				add(ruleAction5, position)
			}
			return true
		},
		/* 32 Action6 <- <{ p.value = buffer[begin:end] }> */
		func() bool {
			{
				add(ruleAction6, position)
			}
			return true
		},
		/* 33 Action7 <- <{ p.addTemplate(buffer[begin:end]) }> */
		func() bool {
			{
				add(ruleAction7, position)
			}
			return true
		},
		/* 34 Action8 <- <{ p.addSample(buffer[begin:end]) }> */
		func() bool {
			{
				add(ruleAction8, position)
			}
			return true
		},
	}
	p.rules = _rules
}
