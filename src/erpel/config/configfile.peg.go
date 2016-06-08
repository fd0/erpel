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
	ruleline
	ruleSection
	ruleStatement
	ruleName
	ruleValue
	ruleSingleQuotedvalue
	ruleDoubleQuotedValue
	ruleRawValue
	ruleComment
	ruleEOF
	ruleEOL
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
	"line",
	"Section",
	"Statement",
	"Name",
	"Value",
	"SingleQuotedvalue",
	"DoubleQuotedValue",
	"RawValue",
	"Comment",
	"EOF",
	"EOL",
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

type erpelParser struct {
	configState

	Buffer string
	buffer []rune
	rules  [21]func() bool
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
			p.newSection(buffer[begin:end])
		case ruleAction1:
			p.set(p.name, p.value)
		case ruleAction2:
			p.name = buffer[begin:end]
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

	_rules = [...]func() bool{
		nil,
		/* 0 start <- <((line EOL)* line? EOF)> */
		func() bool {
			position0, tokenIndex0, depth0 := position, tokenIndex, depth
			{
				position1 := position
				depth++
			l2:
				{
					position3, tokenIndex3, depth3 := position, tokenIndex, depth
					if !_rules[ruleline]() {
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
					if !_rules[ruleline]() {
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
		/* 1 line <- <(S (Comment / Section / Statement)? S)> */
		func() bool {
			position6, tokenIndex6, depth6 := position, tokenIndex, depth
			{
				position7 := position
				depth++
				if !_rules[ruleS]() {
					goto l6
				}
				{
					position8, tokenIndex8, depth8 := position, tokenIndex, depth
					{
						position10, tokenIndex10, depth10 := position, tokenIndex, depth
						if !_rules[ruleComment]() {
							goto l11
						}
						goto l10
					l11:
						position, tokenIndex, depth = position10, tokenIndex10, depth10
						if !_rules[ruleSection]() {
							goto l12
						}
						goto l10
					l12:
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
				if !_rules[ruleS]() {
					goto l6
				}
				depth--
				add(ruleline, position7)
			}
			return true
		l6:
			position, tokenIndex, depth = position6, tokenIndex6, depth6
			return false
		},
		/* 2 Section <- <('[' <(!']' !EOL .)+> ']' Action0)> */
		func() bool {
			position13, tokenIndex13, depth13 := position, tokenIndex, depth
			{
				position14 := position
				depth++
				if buffer[position] != rune('[') {
					goto l13
				}
				position++
				{
					position15 := position
					depth++
					{
						position18, tokenIndex18, depth18 := position, tokenIndex, depth
						if buffer[position] != rune(']') {
							goto l18
						}
						position++
						goto l13
					l18:
						position, tokenIndex, depth = position18, tokenIndex18, depth18
					}
					{
						position19, tokenIndex19, depth19 := position, tokenIndex, depth
						if !_rules[ruleEOL]() {
							goto l19
						}
						goto l13
					l19:
						position, tokenIndex, depth = position19, tokenIndex19, depth19
					}
					if !matchDot() {
						goto l13
					}
				l16:
					{
						position17, tokenIndex17, depth17 := position, tokenIndex, depth
						{
							position20, tokenIndex20, depth20 := position, tokenIndex, depth
							if buffer[position] != rune(']') {
								goto l20
							}
							position++
							goto l17
						l20:
							position, tokenIndex, depth = position20, tokenIndex20, depth20
						}
						{
							position21, tokenIndex21, depth21 := position, tokenIndex, depth
							if !_rules[ruleEOL]() {
								goto l21
							}
							goto l17
						l21:
							position, tokenIndex, depth = position21, tokenIndex21, depth21
						}
						if !matchDot() {
							goto l17
						}
						goto l16
					l17:
						position, tokenIndex, depth = position17, tokenIndex17, depth17
					}
					depth--
					add(rulePegText, position15)
				}
				if buffer[position] != rune(']') {
					goto l13
				}
				position++
				if !_rules[ruleAction0]() {
					goto l13
				}
				depth--
				add(ruleSection, position14)
			}
			return true
		l13:
			position, tokenIndex, depth = position13, tokenIndex13, depth13
			return false
		},
		/* 3 Statement <- <(Name S '=' S Value Action1)> */
		func() bool {
			position22, tokenIndex22, depth22 := position, tokenIndex, depth
			{
				position23 := position
				depth++
				if !_rules[ruleName]() {
					goto l22
				}
				if !_rules[ruleS]() {
					goto l22
				}
				if buffer[position] != rune('=') {
					goto l22
				}
				position++
				if !_rules[ruleS]() {
					goto l22
				}
				if !_rules[ruleValue]() {
					goto l22
				}
				if !_rules[ruleAction1]() {
					goto l22
				}
				depth--
				add(ruleStatement, position23)
			}
			return true
		l22:
			position, tokenIndex, depth = position22, tokenIndex22, depth22
			return false
		},
		/* 4 Name <- <(<(!'=' !EOL .)+> Action2)> */
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
						if buffer[position] != rune('=') {
							goto l29
						}
						position++
						goto l24
					l29:
						position, tokenIndex, depth = position29, tokenIndex29, depth29
					}
					{
						position30, tokenIndex30, depth30 := position, tokenIndex, depth
						if !_rules[ruleEOL]() {
							goto l30
						}
						goto l24
					l30:
						position, tokenIndex, depth = position30, tokenIndex30, depth30
					}
					if !matchDot() {
						goto l24
					}
				l27:
					{
						position28, tokenIndex28, depth28 := position, tokenIndex, depth
						{
							position31, tokenIndex31, depth31 := position, tokenIndex, depth
							if buffer[position] != rune('=') {
								goto l31
							}
							position++
							goto l28
						l31:
							position, tokenIndex, depth = position31, tokenIndex31, depth31
						}
						{
							position32, tokenIndex32, depth32 := position, tokenIndex, depth
							if !_rules[ruleEOL]() {
								goto l32
							}
							goto l28
						l32:
							position, tokenIndex, depth = position32, tokenIndex32, depth32
						}
						if !matchDot() {
							goto l28
						}
						goto l27
					l28:
						position, tokenIndex, depth = position28, tokenIndex28, depth28
					}
					depth--
					add(rulePegText, position26)
				}
				if !_rules[ruleAction2]() {
					goto l24
				}
				depth--
				add(ruleName, position25)
			}
			return true
		l24:
			position, tokenIndex, depth = position24, tokenIndex24, depth24
			return false
		},
		/* 5 Value <- <(DoubleQuotedValue / SingleQuotedvalue / RawValue)> */
		func() bool {
			position33, tokenIndex33, depth33 := position, tokenIndex, depth
			{
				position34 := position
				depth++
				{
					position35, tokenIndex35, depth35 := position, tokenIndex, depth
					if !_rules[ruleDoubleQuotedValue]() {
						goto l36
					}
					goto l35
				l36:
					position, tokenIndex, depth = position35, tokenIndex35, depth35
					if !_rules[ruleSingleQuotedvalue]() {
						goto l37
					}
					goto l35
				l37:
					position, tokenIndex, depth = position35, tokenIndex35, depth35
					if !_rules[ruleRawValue]() {
						goto l33
					}
				}
			l35:
				depth--
				add(ruleValue, position34)
			}
			return true
		l33:
			position, tokenIndex, depth = position33, tokenIndex33, depth33
			return false
		},
		/* 6 SingleQuotedvalue <- <(<('\'' ('\'' / (!EOL !'\'' .))* '\'')> Action3)> */
		func() bool {
			position38, tokenIndex38, depth38 := position, tokenIndex, depth
			{
				position39 := position
				depth++
				{
					position40 := position
					depth++
					if buffer[position] != rune('\'') {
						goto l38
					}
					position++
				l41:
					{
						position42, tokenIndex42, depth42 := position, tokenIndex, depth
						{
							position43, tokenIndex43, depth43 := position, tokenIndex, depth
							if buffer[position] != rune('\'') {
								goto l44
							}
							position++
							goto l43
						l44:
							position, tokenIndex, depth = position43, tokenIndex43, depth43
							{
								position45, tokenIndex45, depth45 := position, tokenIndex, depth
								if !_rules[ruleEOL]() {
									goto l45
								}
								goto l42
							l45:
								position, tokenIndex, depth = position45, tokenIndex45, depth45
							}
							{
								position46, tokenIndex46, depth46 := position, tokenIndex, depth
								if buffer[position] != rune('\'') {
									goto l46
								}
								position++
								goto l42
							l46:
								position, tokenIndex, depth = position46, tokenIndex46, depth46
							}
							if !matchDot() {
								goto l42
							}
						}
					l43:
						goto l41
					l42:
						position, tokenIndex, depth = position42, tokenIndex42, depth42
					}
					if buffer[position] != rune('\'') {
						goto l38
					}
					position++
					depth--
					add(rulePegText, position40)
				}
				if !_rules[ruleAction3]() {
					goto l38
				}
				depth--
				add(ruleSingleQuotedvalue, position39)
			}
			return true
		l38:
			position, tokenIndex, depth = position38, tokenIndex38, depth38
			return false
		},
		/* 7 DoubleQuotedValue <- <(<('"' ('"' / (!EOL !'"' .))* '"')> Action4)> */
		func() bool {
			position47, tokenIndex47, depth47 := position, tokenIndex, depth
			{
				position48 := position
				depth++
				{
					position49 := position
					depth++
					if buffer[position] != rune('"') {
						goto l47
					}
					position++
				l50:
					{
						position51, tokenIndex51, depth51 := position, tokenIndex, depth
						{
							position52, tokenIndex52, depth52 := position, tokenIndex, depth
							if buffer[position] != rune('"') {
								goto l53
							}
							position++
							goto l52
						l53:
							position, tokenIndex, depth = position52, tokenIndex52, depth52
							{
								position54, tokenIndex54, depth54 := position, tokenIndex, depth
								if !_rules[ruleEOL]() {
									goto l54
								}
								goto l51
							l54:
								position, tokenIndex, depth = position54, tokenIndex54, depth54
							}
							{
								position55, tokenIndex55, depth55 := position, tokenIndex, depth
								if buffer[position] != rune('"') {
									goto l55
								}
								position++
								goto l51
							l55:
								position, tokenIndex, depth = position55, tokenIndex55, depth55
							}
							if !matchDot() {
								goto l51
							}
						}
					l52:
						goto l50
					l51:
						position, tokenIndex, depth = position51, tokenIndex51, depth51
					}
					if buffer[position] != rune('"') {
						goto l47
					}
					position++
					depth--
					add(rulePegText, position49)
				}
				if !_rules[ruleAction4]() {
					goto l47
				}
				depth--
				add(ruleDoubleQuotedValue, position48)
			}
			return true
		l47:
			position, tokenIndex, depth = position47, tokenIndex47, depth47
			return false
		},
		/* 8 RawValue <- <(<(!EOL .)*> Action5)> */
		func() bool {
			position56, tokenIndex56, depth56 := position, tokenIndex, depth
			{
				position57 := position
				depth++
				{
					position58 := position
					depth++
				l59:
					{
						position60, tokenIndex60, depth60 := position, tokenIndex, depth
						{
							position61, tokenIndex61, depth61 := position, tokenIndex, depth
							if !_rules[ruleEOL]() {
								goto l61
							}
							goto l60
						l61:
							position, tokenIndex, depth = position61, tokenIndex61, depth61
						}
						if !matchDot() {
							goto l60
						}
						goto l59
					l60:
						position, tokenIndex, depth = position60, tokenIndex60, depth60
					}
					depth--
					add(rulePegText, position58)
				}
				if !_rules[ruleAction5]() {
					goto l56
				}
				depth--
				add(ruleRawValue, position57)
			}
			return true
		l56:
			position, tokenIndex, depth = position56, tokenIndex56, depth56
			return false
		},
		/* 9 Comment <- <(S '#' (!EOL .)*)> */
		func() bool {
			position62, tokenIndex62, depth62 := position, tokenIndex, depth
			{
				position63 := position
				depth++
				if !_rules[ruleS]() {
					goto l62
				}
				if buffer[position] != rune('#') {
					goto l62
				}
				position++
			l64:
				{
					position65, tokenIndex65, depth65 := position, tokenIndex, depth
					{
						position66, tokenIndex66, depth66 := position, tokenIndex, depth
						if !_rules[ruleEOL]() {
							goto l66
						}
						goto l65
					l66:
						position, tokenIndex, depth = position66, tokenIndex66, depth66
					}
					if !matchDot() {
						goto l65
					}
					goto l64
				l65:
					position, tokenIndex, depth = position65, tokenIndex65, depth65
				}
				depth--
				add(ruleComment, position63)
			}
			return true
		l62:
			position, tokenIndex, depth = position62, tokenIndex62, depth62
			return false
		},
		/* 10 EOF <- <!.> */
		func() bool {
			position67, tokenIndex67, depth67 := position, tokenIndex, depth
			{
				position68 := position
				depth++
				{
					position69, tokenIndex69, depth69 := position, tokenIndex, depth
					if !matchDot() {
						goto l69
					}
					goto l67
				l69:
					position, tokenIndex, depth = position69, tokenIndex69, depth69
				}
				depth--
				add(ruleEOF, position68)
			}
			return true
		l67:
			position, tokenIndex, depth = position67, tokenIndex67, depth67
			return false
		},
		/* 11 EOL <- <('\r' / '\n')> */
		func() bool {
			position70, tokenIndex70, depth70 := position, tokenIndex, depth
			{
				position71 := position
				depth++
				{
					position72, tokenIndex72, depth72 := position, tokenIndex, depth
					if buffer[position] != rune('\r') {
						goto l73
					}
					position++
					goto l72
				l73:
					position, tokenIndex, depth = position72, tokenIndex72, depth72
					if buffer[position] != rune('\n') {
						goto l70
					}
					position++
				}
			l72:
				depth--
				add(ruleEOL, position71)
			}
			return true
		l70:
			position, tokenIndex, depth = position70, tokenIndex70, depth70
			return false
		},
		/* 12 S <- <(' ' / '\t')*> */
		func() bool {
			{
				position75 := position
				depth++
			l76:
				{
					position77, tokenIndex77, depth77 := position, tokenIndex, depth
					{
						position78, tokenIndex78, depth78 := position, tokenIndex, depth
						if buffer[position] != rune(' ') {
							goto l79
						}
						position++
						goto l78
					l79:
						position, tokenIndex, depth = position78, tokenIndex78, depth78
						if buffer[position] != rune('\t') {
							goto l77
						}
						position++
					}
				l78:
					goto l76
				l77:
					position, tokenIndex, depth = position77, tokenIndex77, depth77
				}
				depth--
				add(ruleS, position75)
			}
			return true
		},
		nil,
		/* 15 Action0 <- <{ p.newSection(buffer[begin:end]) }> */
		func() bool {
			{
				add(ruleAction0, position)
			}
			return true
		},
		/* 16 Action1 <- <{ p.set(p.name, p.value) }> */
		func() bool {
			{
				add(ruleAction1, position)
			}
			return true
		},
		/* 17 Action2 <- <{ p.name = buffer[begin:end] }> */
		func() bool {
			{
				add(ruleAction2, position)
			}
			return true
		},
		/* 18 Action3 <- <{ p.value = buffer[begin:end] }> */
		func() bool {
			{
				add(ruleAction3, position)
			}
			return true
		},
		/* 19 Action4 <- <{ p.value = buffer[begin:end] }> */
		func() bool {
			{
				add(ruleAction4, position)
			}
			return true
		},
		/* 20 Action5 <- <{ p.value = buffer[begin:end] }> */
		func() bool {
			{
				add(ruleAction5, position)
			}
			return true
		},
	}
	p.rules = _rules
}
