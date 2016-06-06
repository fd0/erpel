package config

import (
	"fmt"
	"math"
	"sort"
	"strconv"
)

const end_symbol rune = 1114112

/* The rule types inferred from the grammar are below. */
type pegRule uint8

const (
	ruleUnknown pegRule = iota
	rulestart
	ruleline
	ruleStatement
	ruleName
	ruleValue
	ruleSingleQuotedvalue
	ruleDoubleQuotedValue
	ruleRawValue
	ruleComment
	ruleEOF
	ruleEOL
	rule_
	ruleAction0
	rulePegText
	ruleAction1
	ruleAction2
	ruleAction3
	ruleAction4

	rulePre_
	rule_In_
	rule_Suf
)

var rul3s = [...]string{
	"Unknown",
	"start",
	"line",
	"Statement",
	"Name",
	"Value",
	"SingleQuotedvalue",
	"DoubleQuotedValue",
	"RawValue",
	"Comment",
	"EOF",
	"EOL",
	"_",
	"Action0",
	"PegText",
	"Action1",
	"Action2",
	"Action3",
	"Action4",

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

func (ast *node32) Print(buffer string) {
	ast.print(0, buffer)
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
		for i, _ := range states {
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
							write(token32{pegRule: rule_In_, begin: c.end, end: b.begin}, true)
						}
						break
					}
				}

				if a.begin < b.begin {
					write(token32{pegRule: rulePre_, begin: a.begin, end: b.begin}, true)
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
					write(token32{pegRule: rule_Suf, begin: b.end, end: a.end}, true)
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
	for i, _ := range tokens {
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
	rules  [19]func() bool
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
			p.set(p.name, p.value)
		case ruleAction1:
			p.name = buffer[begin:end]
		case ruleAction2:
			p.value = buffer[begin:end]
		case ruleAction3:
			p.value = buffer[begin:end]
		case ruleAction4:
			p.value = buffer[begin:end]

		}
	}
	_, _, _, _, _ = buffer, _buffer, text, begin, end
}

func (p *erpelParser) Init() {
	p.buffer = []rune(p.Buffer)
	if len(p.buffer) == 0 || p.buffer[len(p.buffer)-1] != end_symbol {
		p.buffer = append(p.buffer, end_symbol)
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
		if buffer[position] != end_symbol {
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
		/* 1 line <- <(_ (Comment / Statement)? _)> */
		func() bool {
			position6, tokenIndex6, depth6 := position, tokenIndex, depth
			{
				position7 := position
				depth++
				if !_rules[rule_]() {
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
				if !_rules[rule_]() {
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
		/* 2 Statement <- <(Name _ '=' _ Value Action0)> */
		func() bool {
			position12, tokenIndex12, depth12 := position, tokenIndex, depth
			{
				position13 := position
				depth++
				if !_rules[ruleName]() {
					goto l12
				}
				if !_rules[rule_]() {
					goto l12
				}
				if buffer[position] != rune('=') {
					goto l12
				}
				position++
				if !_rules[rule_]() {
					goto l12
				}
				if !_rules[ruleValue]() {
					goto l12
				}
				if !_rules[ruleAction0]() {
					goto l12
				}
				depth--
				add(ruleStatement, position13)
			}
			return true
		l12:
			position, tokenIndex, depth = position12, tokenIndex12, depth12
			return false
		},
		/* 3 Name <- <(<(!'=' .)+> Action1)> */
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
						if buffer[position] != rune('=') {
							goto l19
						}
						position++
						goto l14
					l19:
						position, tokenIndex, depth = position19, tokenIndex19, depth19
					}
					if !matchDot() {
						goto l14
					}
				l17:
					{
						position18, tokenIndex18, depth18 := position, tokenIndex, depth
						{
							position20, tokenIndex20, depth20 := position, tokenIndex, depth
							if buffer[position] != rune('=') {
								goto l20
							}
							position++
							goto l18
						l20:
							position, tokenIndex, depth = position20, tokenIndex20, depth20
						}
						if !matchDot() {
							goto l18
						}
						goto l17
					l18:
						position, tokenIndex, depth = position18, tokenIndex18, depth18
					}
					depth--
					add(rulePegText, position16)
				}
				if !_rules[ruleAction1]() {
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
		/* 4 Value <- <(DoubleQuotedValue / SingleQuotedvalue / RawValue)> */
		func() bool {
			position21, tokenIndex21, depth21 := position, tokenIndex, depth
			{
				position22 := position
				depth++
				{
					position23, tokenIndex23, depth23 := position, tokenIndex, depth
					if !_rules[ruleDoubleQuotedValue]() {
						goto l24
					}
					goto l23
				l24:
					position, tokenIndex, depth = position23, tokenIndex23, depth23
					if !_rules[ruleSingleQuotedvalue]() {
						goto l25
					}
					goto l23
				l25:
					position, tokenIndex, depth = position23, tokenIndex23, depth23
					if !_rules[ruleRawValue]() {
						goto l21
					}
				}
			l23:
				depth--
				add(ruleValue, position22)
			}
			return true
		l21:
			position, tokenIndex, depth = position21, tokenIndex21, depth21
			return false
		},
		/* 5 SingleQuotedvalue <- <(<('\'' ('\'' / (!EOL !'\'' .))* '\'')> Action2)> */
		func() bool {
			position26, tokenIndex26, depth26 := position, tokenIndex, depth
			{
				position27 := position
				depth++
				{
					position28 := position
					depth++
					if buffer[position] != rune('\'') {
						goto l26
					}
					position++
				l29:
					{
						position30, tokenIndex30, depth30 := position, tokenIndex, depth
						{
							position31, tokenIndex31, depth31 := position, tokenIndex, depth
							if buffer[position] != rune('\'') {
								goto l32
							}
							position++
							goto l31
						l32:
							position, tokenIndex, depth = position31, tokenIndex31, depth31
							{
								position33, tokenIndex33, depth33 := position, tokenIndex, depth
								if !_rules[ruleEOL]() {
									goto l33
								}
								goto l30
							l33:
								position, tokenIndex, depth = position33, tokenIndex33, depth33
							}
							{
								position34, tokenIndex34, depth34 := position, tokenIndex, depth
								if buffer[position] != rune('\'') {
									goto l34
								}
								position++
								goto l30
							l34:
								position, tokenIndex, depth = position34, tokenIndex34, depth34
							}
							if !matchDot() {
								goto l30
							}
						}
					l31:
						goto l29
					l30:
						position, tokenIndex, depth = position30, tokenIndex30, depth30
					}
					if buffer[position] != rune('\'') {
						goto l26
					}
					position++
					depth--
					add(rulePegText, position28)
				}
				if !_rules[ruleAction2]() {
					goto l26
				}
				depth--
				add(ruleSingleQuotedvalue, position27)
			}
			return true
		l26:
			position, tokenIndex, depth = position26, tokenIndex26, depth26
			return false
		},
		/* 6 DoubleQuotedValue <- <(<('"' ('"' / (!EOL !'"' .))* '"')> Action3)> */
		func() bool {
			position35, tokenIndex35, depth35 := position, tokenIndex, depth
			{
				position36 := position
				depth++
				{
					position37 := position
					depth++
					if buffer[position] != rune('"') {
						goto l35
					}
					position++
				l38:
					{
						position39, tokenIndex39, depth39 := position, tokenIndex, depth
						{
							position40, tokenIndex40, depth40 := position, tokenIndex, depth
							if buffer[position] != rune('"') {
								goto l41
							}
							position++
							goto l40
						l41:
							position, tokenIndex, depth = position40, tokenIndex40, depth40
							{
								position42, tokenIndex42, depth42 := position, tokenIndex, depth
								if !_rules[ruleEOL]() {
									goto l42
								}
								goto l39
							l42:
								position, tokenIndex, depth = position42, tokenIndex42, depth42
							}
							{
								position43, tokenIndex43, depth43 := position, tokenIndex, depth
								if buffer[position] != rune('"') {
									goto l43
								}
								position++
								goto l39
							l43:
								position, tokenIndex, depth = position43, tokenIndex43, depth43
							}
							if !matchDot() {
								goto l39
							}
						}
					l40:
						goto l38
					l39:
						position, tokenIndex, depth = position39, tokenIndex39, depth39
					}
					if buffer[position] != rune('"') {
						goto l35
					}
					position++
					depth--
					add(rulePegText, position37)
				}
				if !_rules[ruleAction3]() {
					goto l35
				}
				depth--
				add(ruleDoubleQuotedValue, position36)
			}
			return true
		l35:
			position, tokenIndex, depth = position35, tokenIndex35, depth35
			return false
		},
		/* 7 RawValue <- <(<(!EOL .)*> Action4)> */
		func() bool {
			position44, tokenIndex44, depth44 := position, tokenIndex, depth
			{
				position45 := position
				depth++
				{
					position46 := position
					depth++
				l47:
					{
						position48, tokenIndex48, depth48 := position, tokenIndex, depth
						{
							position49, tokenIndex49, depth49 := position, tokenIndex, depth
							if !_rules[ruleEOL]() {
								goto l49
							}
							goto l48
						l49:
							position, tokenIndex, depth = position49, tokenIndex49, depth49
						}
						if !matchDot() {
							goto l48
						}
						goto l47
					l48:
						position, tokenIndex, depth = position48, tokenIndex48, depth48
					}
					depth--
					add(rulePegText, position46)
				}
				if !_rules[ruleAction4]() {
					goto l44
				}
				depth--
				add(ruleRawValue, position45)
			}
			return true
		l44:
			position, tokenIndex, depth = position44, tokenIndex44, depth44
			return false
		},
		/* 8 Comment <- <(_ '#' (!EOL .)*)> */
		func() bool {
			position50, tokenIndex50, depth50 := position, tokenIndex, depth
			{
				position51 := position
				depth++
				if !_rules[rule_]() {
					goto l50
				}
				if buffer[position] != rune('#') {
					goto l50
				}
				position++
			l52:
				{
					position53, tokenIndex53, depth53 := position, tokenIndex, depth
					{
						position54, tokenIndex54, depth54 := position, tokenIndex, depth
						if !_rules[ruleEOL]() {
							goto l54
						}
						goto l53
					l54:
						position, tokenIndex, depth = position54, tokenIndex54, depth54
					}
					if !matchDot() {
						goto l53
					}
					goto l52
				l53:
					position, tokenIndex, depth = position53, tokenIndex53, depth53
				}
				depth--
				add(ruleComment, position51)
			}
			return true
		l50:
			position, tokenIndex, depth = position50, tokenIndex50, depth50
			return false
		},
		/* 9 EOF <- <!.> */
		func() bool {
			position55, tokenIndex55, depth55 := position, tokenIndex, depth
			{
				position56 := position
				depth++
				{
					position57, tokenIndex57, depth57 := position, tokenIndex, depth
					if !matchDot() {
						goto l57
					}
					goto l55
				l57:
					position, tokenIndex, depth = position57, tokenIndex57, depth57
				}
				depth--
				add(ruleEOF, position56)
			}
			return true
		l55:
			position, tokenIndex, depth = position55, tokenIndex55, depth55
			return false
		},
		/* 10 EOL <- <('\r' / '\n')> */
		func() bool {
			position58, tokenIndex58, depth58 := position, tokenIndex, depth
			{
				position59 := position
				depth++
				{
					position60, tokenIndex60, depth60 := position, tokenIndex, depth
					if buffer[position] != rune('\r') {
						goto l61
					}
					position++
					goto l60
				l61:
					position, tokenIndex, depth = position60, tokenIndex60, depth60
					if buffer[position] != rune('\n') {
						goto l58
					}
					position++
				}
			l60:
				depth--
				add(ruleEOL, position59)
			}
			return true
		l58:
			position, tokenIndex, depth = position58, tokenIndex58, depth58
			return false
		},
		/* 11 _ <- <(' ' / '\t')*> */
		func() bool {
			{
				position63 := position
				depth++
			l64:
				{
					position65, tokenIndex65, depth65 := position, tokenIndex, depth
					{
						position66, tokenIndex66, depth66 := position, tokenIndex, depth
						if buffer[position] != rune(' ') {
							goto l67
						}
						position++
						goto l66
					l67:
						position, tokenIndex, depth = position66, tokenIndex66, depth66
						if buffer[position] != rune('\t') {
							goto l65
						}
						position++
					}
				l66:
					goto l64
				l65:
					position, tokenIndex, depth = position65, tokenIndex65, depth65
				}
				depth--
				add(rule_, position63)
			}
			return true
		},
		/* 13 Action0 <- <{ p.set(p.name, p.value) }> */
		func() bool {
			{
				add(ruleAction0, position)
			}
			return true
		},
		nil,
		/* 15 Action1 <- <{ p.name = buffer[begin:end] }> */
		func() bool {
			{
				add(ruleAction1, position)
			}
			return true
		},
		/* 16 Action2 <- <{ p.value = buffer[begin:end] }> */
		func() bool {
			{
				add(ruleAction2, position)
			}
			return true
		},
		/* 17 Action3 <- <{ p.value = buffer[begin:end] }> */
		func() bool {
			{
				add(ruleAction3, position)
			}
			return true
		},
		/* 18 Action4 <- <{ p.value = buffer[begin:end] }> */
		func() bool {
			{
				add(ruleAction4, position)
			}
			return true
		},
	}
	p.rules = _rules
}
