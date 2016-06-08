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
	ruleValue
	ruleSingleQuotedvalue
	ruleDoubleQuotedValue
	ruleRawValue
	ruleComment
	ruleEOF
	ruleEOL
	rules
	ruleS
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
	"Value",
	"SingleQuotedvalue",
	"DoubleQuotedValue",
	"RawValue",
	"Comment",
	"EOF",
	"EOL",
	"s",
	"S",
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
	rules  [25]func() bool
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
		/* 5 Statement <- <(Name s '=' s Value Action2)> */
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
				if !_rules[ruleValue]() {
					goto l21
				}
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
		/* 6 Name <- <(<([a-z] / [A-Z] / [0-9] / '_')+> Action3)> */
		func() bool {
			position23, tokenIndex23, depth23 := position, tokenIndex, depth
			{
				position24 := position
				depth++
				{
					position25 := position
					depth++
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
						if buffer[position] != rune('_') {
							goto l23
						}
						position++
					}
				l28:
				l26:
					{
						position27, tokenIndex27, depth27 := position, tokenIndex, depth
						{
							position32, tokenIndex32, depth32 := position, tokenIndex, depth
							if c := buffer[position]; c < rune('a') || c > rune('z') {
								goto l33
							}
							position++
							goto l32
						l33:
							position, tokenIndex, depth = position32, tokenIndex32, depth32
							if c := buffer[position]; c < rune('A') || c > rune('Z') {
								goto l34
							}
							position++
							goto l32
						l34:
							position, tokenIndex, depth = position32, tokenIndex32, depth32
							if c := buffer[position]; c < rune('0') || c > rune('9') {
								goto l35
							}
							position++
							goto l32
						l35:
							position, tokenIndex, depth = position32, tokenIndex32, depth32
							if buffer[position] != rune('_') {
								goto l27
							}
							position++
						}
					l32:
						goto l26
					l27:
						position, tokenIndex, depth = position27, tokenIndex27, depth27
					}
					depth--
					add(rulePegText, position25)
				}
				if !_rules[ruleAction3]() {
					goto l23
				}
				depth--
				add(ruleName, position24)
			}
			return true
		l23:
			position, tokenIndex, depth = position23, tokenIndex23, depth23
			return false
		},
		/* 7 Value <- <(DoubleQuotedValue / SingleQuotedvalue / RawValue)> */
		func() bool {
			position36, tokenIndex36, depth36 := position, tokenIndex, depth
			{
				position37 := position
				depth++
				{
					position38, tokenIndex38, depth38 := position, tokenIndex, depth
					if !_rules[ruleDoubleQuotedValue]() {
						goto l39
					}
					goto l38
				l39:
					position, tokenIndex, depth = position38, tokenIndex38, depth38
					if !_rules[ruleSingleQuotedvalue]() {
						goto l40
					}
					goto l38
				l40:
					position, tokenIndex, depth = position38, tokenIndex38, depth38
					if !_rules[ruleRawValue]() {
						goto l36
					}
				}
			l38:
				depth--
				add(ruleValue, position37)
			}
			return true
		l36:
			position, tokenIndex, depth = position36, tokenIndex36, depth36
			return false
		},
		/* 8 SingleQuotedvalue <- <(<('\'' ('\'' / (!EOL !'\'' .))* '\'')> Action4)> */
		func() bool {
			position41, tokenIndex41, depth41 := position, tokenIndex, depth
			{
				position42 := position
				depth++
				{
					position43 := position
					depth++
					if buffer[position] != rune('\'') {
						goto l41
					}
					position++
				l44:
					{
						position45, tokenIndex45, depth45 := position, tokenIndex, depth
						{
							position46, tokenIndex46, depth46 := position, tokenIndex, depth
							if buffer[position] != rune('\'') {
								goto l47
							}
							position++
							goto l46
						l47:
							position, tokenIndex, depth = position46, tokenIndex46, depth46
							{
								position48, tokenIndex48, depth48 := position, tokenIndex, depth
								if !_rules[ruleEOL]() {
									goto l48
								}
								goto l45
							l48:
								position, tokenIndex, depth = position48, tokenIndex48, depth48
							}
							{
								position49, tokenIndex49, depth49 := position, tokenIndex, depth
								if buffer[position] != rune('\'') {
									goto l49
								}
								position++
								goto l45
							l49:
								position, tokenIndex, depth = position49, tokenIndex49, depth49
							}
							if !matchDot() {
								goto l45
							}
						}
					l46:
						goto l44
					l45:
						position, tokenIndex, depth = position45, tokenIndex45, depth45
					}
					if buffer[position] != rune('\'') {
						goto l41
					}
					position++
					depth--
					add(rulePegText, position43)
				}
				if !_rules[ruleAction4]() {
					goto l41
				}
				depth--
				add(ruleSingleQuotedvalue, position42)
			}
			return true
		l41:
			position, tokenIndex, depth = position41, tokenIndex41, depth41
			return false
		},
		/* 9 DoubleQuotedValue <- <(<('"' ('"' / (!EOL !'"' .))* '"')> Action5)> */
		func() bool {
			position50, tokenIndex50, depth50 := position, tokenIndex, depth
			{
				position51 := position
				depth++
				{
					position52 := position
					depth++
					if buffer[position] != rune('"') {
						goto l50
					}
					position++
				l53:
					{
						position54, tokenIndex54, depth54 := position, tokenIndex, depth
						{
							position55, tokenIndex55, depth55 := position, tokenIndex, depth
							if buffer[position] != rune('"') {
								goto l56
							}
							position++
							goto l55
						l56:
							position, tokenIndex, depth = position55, tokenIndex55, depth55
							{
								position57, tokenIndex57, depth57 := position, tokenIndex, depth
								if !_rules[ruleEOL]() {
									goto l57
								}
								goto l54
							l57:
								position, tokenIndex, depth = position57, tokenIndex57, depth57
							}
							{
								position58, tokenIndex58, depth58 := position, tokenIndex, depth
								if buffer[position] != rune('"') {
									goto l58
								}
								position++
								goto l54
							l58:
								position, tokenIndex, depth = position58, tokenIndex58, depth58
							}
							if !matchDot() {
								goto l54
							}
						}
					l55:
						goto l53
					l54:
						position, tokenIndex, depth = position54, tokenIndex54, depth54
					}
					if buffer[position] != rune('"') {
						goto l50
					}
					position++
					depth--
					add(rulePegText, position52)
				}
				if !_rules[ruleAction5]() {
					goto l50
				}
				depth--
				add(ruleDoubleQuotedValue, position51)
			}
			return true
		l50:
			position, tokenIndex, depth = position50, tokenIndex50, depth50
			return false
		},
		/* 10 RawValue <- <(<(!EOL .)*> Action6)> */
		func() bool {
			position59, tokenIndex59, depth59 := position, tokenIndex, depth
			{
				position60 := position
				depth++
				{
					position61 := position
					depth++
				l62:
					{
						position63, tokenIndex63, depth63 := position, tokenIndex, depth
						{
							position64, tokenIndex64, depth64 := position, tokenIndex, depth
							if !_rules[ruleEOL]() {
								goto l64
							}
							goto l63
						l64:
							position, tokenIndex, depth = position64, tokenIndex64, depth64
						}
						if !matchDot() {
							goto l63
						}
						goto l62
					l63:
						position, tokenIndex, depth = position63, tokenIndex63, depth63
					}
					depth--
					add(rulePegText, position61)
				}
				if !_rules[ruleAction6]() {
					goto l59
				}
				depth--
				add(ruleRawValue, position60)
			}
			return true
		l59:
			position, tokenIndex, depth = position59, tokenIndex59, depth59
			return false
		},
		/* 11 Comment <- <(s '#' (!EOL .)*)> */
		func() bool {
			position65, tokenIndex65, depth65 := position, tokenIndex, depth
			{
				position66 := position
				depth++
				if !_rules[rules]() {
					goto l65
				}
				if buffer[position] != rune('#') {
					goto l65
				}
				position++
			l67:
				{
					position68, tokenIndex68, depth68 := position, tokenIndex, depth
					{
						position69, tokenIndex69, depth69 := position, tokenIndex, depth
						if !_rules[ruleEOL]() {
							goto l69
						}
						goto l68
					l69:
						position, tokenIndex, depth = position69, tokenIndex69, depth69
					}
					if !matchDot() {
						goto l68
					}
					goto l67
				l68:
					position, tokenIndex, depth = position68, tokenIndex68, depth68
				}
				depth--
				add(ruleComment, position66)
			}
			return true
		l65:
			position, tokenIndex, depth = position65, tokenIndex65, depth65
			return false
		},
		/* 12 EOF <- <!.> */
		func() bool {
			position70, tokenIndex70, depth70 := position, tokenIndex, depth
			{
				position71 := position
				depth++
				{
					position72, tokenIndex72, depth72 := position, tokenIndex, depth
					if !matchDot() {
						goto l72
					}
					goto l70
				l72:
					position, tokenIndex, depth = position72, tokenIndex72, depth72
				}
				depth--
				add(ruleEOF, position71)
			}
			return true
		l70:
			position, tokenIndex, depth = position70, tokenIndex70, depth70
			return false
		},
		/* 13 EOL <- <('\r' / '\n')> */
		func() bool {
			position73, tokenIndex73, depth73 := position, tokenIndex, depth
			{
				position74 := position
				depth++
				{
					position75, tokenIndex75, depth75 := position, tokenIndex, depth
					if buffer[position] != rune('\r') {
						goto l76
					}
					position++
					goto l75
				l76:
					position, tokenIndex, depth = position75, tokenIndex75, depth75
					if buffer[position] != rune('\n') {
						goto l73
					}
					position++
				}
			l75:
				depth--
				add(ruleEOL, position74)
			}
			return true
		l73:
			position, tokenIndex, depth = position73, tokenIndex73, depth73
			return false
		},
		/* 14 s <- <(' ' / '\t')*> */
		func() bool {
			{
				position78 := position
				depth++
			l79:
				{
					position80, tokenIndex80, depth80 := position, tokenIndex, depth
					{
						position81, tokenIndex81, depth81 := position, tokenIndex, depth
						if buffer[position] != rune(' ') {
							goto l82
						}
						position++
						goto l81
					l82:
						position, tokenIndex, depth = position81, tokenIndex81, depth81
						if buffer[position] != rune('\t') {
							goto l80
						}
						position++
					}
				l81:
					goto l79
				l80:
					position, tokenIndex, depth = position80, tokenIndex80, depth80
				}
				depth--
				add(rules, position78)
			}
			return true
		},
		/* 15 S <- <(' ' / '\t' / '\r' / '\n')*> */
		nil,
		/* 17 Action0 <- <{ p.setDefaultSection() }> */
		func() bool {
			{
				add(ruleAction0, position)
			}
			return true
		},
		nil,
		/* 19 Action1 <- <{ p.newSection(buffer[begin:end]) }> */
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
		/* 21 Action3 <- <{ p.name = buffer[begin:end] }> */
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
		/* 24 Action6 <- <{ p.value = buffer[begin:end] }> */
		func() bool {
			{
				add(ruleAction6, position)
			}
			return true
		},
	}
	p.rules = _rules
}
