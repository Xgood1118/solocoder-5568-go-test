package filter

import (
	"apitester/internal/models"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type TokenType int

const (
	TokenEOF TokenType = iota
	TokenIdentifier
	TokenNumber
	TokenString
	TokenBool
	TokenOpEQ
	TokenOpNE
	TokenOpLT
	TokenOpGT
	TokenOpLE
	TokenOpGE
	TokenOpAnd
	TokenOpOr
	TokenOpNot
	TokenOpContains
	TokenOpMatches
	TokenLParen
	TokenRParen
)

type Token struct {
	Type  TokenType
	Value string
	Pos   int
}

type ExprNode interface {
	String() string
}

type BinaryExpr struct {
	Op    TokenType
	Left  ExprNode
	Right ExprNode
}

func (b *BinaryExpr) String() string {
	return fmt.Sprintf("(%s %s %s)", b.Left, tokenOpStr(b.Op), b.Right)
}

type UnaryExpr struct {
	Op       TokenType
	Operand  ExprNode
}

func (u *UnaryExpr) String() string {
	return fmt.Sprintf("%s%s", tokenOpStr(u.Op), u.Operand)
}

type FieldExpr struct {
	Name string
}

func (f *FieldExpr) String() string {
	return f.Name
}

type LiteralExpr struct {
	Type  TokenType
	Value string
}

func (l *LiteralExpr) String() string {
	if l.Type == TokenString {
		return fmt.Sprintf("\"%s\"", l.Value)
	}
	return l.Value
}

type ExpressionParser struct {
	validFields map[string]bool
}

func NewExpressionParser() *ExpressionParser {
	return &ExpressionParser{
		validFields: map[string]bool{
			"status":   true,
			"latency":  true,
			"duration": true,
			"name":     true,
			"id":       true,
			"tag":      true,
			"passed":   true,
			"failed":   true,
			"skipped":  true,
		},
	}
}

func (p *ExpressionParser) ParseExpression(expr string) (ExprNode, error) {
	tokens, err := p.tokenize(expr)
	if err != nil {
		return nil, err
	}

	parser := &exprParser{
		tokens: tokens,
		pos:    0,
		fields: p.validFields,
	}

	node, err := parser.parseOr()
	if err != nil {
		return nil, err
	}

	if parser.pos < len(parser.tokens) && parser.tokens[parser.pos].Type != TokenEOF {
		return nil, fmt.Errorf("unexpected token at position %d: %s", parser.tokens[parser.pos].Pos, parser.tokens[parser.pos].Value)
	}

	return node, nil
}

func (p *ExpressionParser) tokenize(expr string) ([]Token, error) {
	var tokens []Token
	pos := 0
	exprLen := len(expr)

	for pos < exprLen {
		ch := expr[pos]

		if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {
			pos++
			continue
		}

		if pos+1 < exprLen {
			twoChar := expr[pos : pos+2]
			switch twoChar {
			case "==":
				tokens = append(tokens, Token{Type: TokenOpEQ, Value: "==", Pos: pos})
				pos += 2
				continue
			case "!=":
				tokens = append(tokens, Token{Type: TokenOpNE, Value: "!=", Pos: pos})
				pos += 2
				continue
			case "<=":
				tokens = append(tokens, Token{Type: TokenOpLE, Value: "<=", Pos: pos})
				pos += 2
				continue
			case ">=":
				tokens = append(tokens, Token{Type: TokenOpGE, Value: ">=", Pos: pos})
				pos += 2
				continue
			case "&&":
				tokens = append(tokens, Token{Type: TokenOpAnd, Value: "&&", Pos: pos})
				pos += 2
				continue
			case "||":
				tokens = append(tokens, Token{Type: TokenOpOr, Value: "||", Pos: pos})
				pos += 2
				continue
			}
		}

		switch ch {
		case '<':
			tokens = append(tokens, Token{Type: TokenOpLT, Value: "<", Pos: pos})
			pos++
			continue
		case '>':
			tokens = append(tokens, Token{Type: TokenOpGT, Value: ">", Pos: pos})
			pos++
			continue
		case '!':
			tokens = append(tokens, Token{Type: TokenOpNot, Value: "!", Pos: pos})
			pos++
			continue
		case '(':
			tokens = append(tokens, Token{Type: TokenLParen, Value: "(", Pos: pos})
			pos++
			continue
		case ')':
			tokens = append(tokens, Token{Type: TokenRParen, Value: ")", Pos: pos})
			pos++
			continue
		case '"', '\'':
			quote := ch
			pos++
			start := pos
			for pos < exprLen && expr[pos] != quote {
				if expr[pos] == '\\' && pos+1 < exprLen {
					pos += 2
					continue
				}
				pos++
			}
			if pos >= exprLen {
				return nil, fmt.Errorf("unterminated string at position %d", start-1)
			}
			value := expr[start:pos]
			value = strings.ReplaceAll(value, `\"`, `"`)
			value = strings.ReplaceAll(value, `\'`, "'")
			tokens = append(tokens, Token{Type: TokenString, Value: value, Pos: start - 1})
			pos++
			continue
		}

		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_' {
			start := pos
			for pos < exprLen && ((expr[pos] >= 'a' && expr[pos] <= 'z') || (expr[pos] >= 'A' && expr[pos] <= 'Z') || (expr[pos] >= '0' && expr[pos] <= '9') || expr[pos] == '_') {
				pos++
			}
			ident := expr[start:pos]
			lowerIdent := strings.ToLower(ident)

			switch lowerIdent {
			case "contains":
				tokens = append(tokens, Token{Type: TokenOpContains, Value: "contains", Pos: start})
			case "matches":
				tokens = append(tokens, Token{Type: TokenOpMatches, Value: "matches", Pos: start})
			case "true":
				tokens = append(tokens, Token{Type: TokenBool, Value: "true", Pos: start})
			case "false":
				tokens = append(tokens, Token{Type: TokenBool, Value: "false", Pos: start})
			default:
				tokens = append(tokens, Token{Type: TokenIdentifier, Value: ident, Pos: start})
			}
			continue
		}

		if (ch >= '0' && ch <= '9') || ch == '.' {
			start := pos
			hasDot := false
			for pos < exprLen && ((expr[pos] >= '0' && expr[pos] <= '9') || expr[pos] == '.') {
				if expr[pos] == '.' {
					if hasDot {
						return nil, fmt.Errorf("invalid number at position %d", start)
					}
					hasDot = true
				}
				pos++
			}
			tokens = append(tokens, Token{Type: TokenNumber, Value: expr[start:pos], Pos: start})
			continue
		}

		return nil, fmt.Errorf("unexpected character '%c' at position %d", ch, pos)
	}

	tokens = append(tokens, Token{Type: TokenEOF, Value: "", Pos: pos})
	return tokens, nil
}

type exprParser struct {
	tokens []Token
	pos    int
	fields map[string]bool
}

func (p *exprParser) peek() Token {
	return p.tokens[p.pos]
}

func (p *exprParser) consume() Token {
	t := p.tokens[p.pos]
	p.pos++
	return t
}

func (p *exprParser) parseOr() (ExprNode, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}

	for p.peek().Type == TokenOpOr {
		op := p.consume()
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{Op: op.Type, Left: left, Right: right}
	}

	return left, nil
}

func (p *exprParser) parseAnd() (ExprNode, error) {
	left, err := p.parseNot()
	if err != nil {
		return nil, err
	}

	for p.peek().Type == TokenOpAnd {
		op := p.consume()
		right, err := p.parseNot()
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{Op: op.Type, Left: left, Right: right}
	}

	return left, nil
}

func (p *exprParser) parseNot() (ExprNode, error) {
	if p.peek().Type == TokenOpNot {
		op := p.consume()
		operand, err := p.parseNot()
		if err != nil {
			return nil, err
		}
		return &UnaryExpr{Op: op.Type, Operand: operand}, nil
	}

	return p.parseComparison()
}

func (p *exprParser) parseComparison() (ExprNode, error) {
	left, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}

	compOps := map[TokenType]bool{
		TokenOpEQ:       true,
		TokenOpNE:       true,
		TokenOpLT:       true,
		TokenOpGT:       true,
		TokenOpLE:       true,
		TokenOpGE:       true,
		TokenOpContains: true,
		TokenOpMatches:  true,
	}

	if compOps[p.peek().Type] {
		op := p.consume()
		right, err := p.parsePrimary()
		if err != nil {
			return nil, err
		}
		return &BinaryExpr{Op: op.Type, Left: left, Right: right}, nil
	}

	return left, nil
}

func (p *exprParser) parsePrimary() (ExprNode, error) {
	tok := p.peek()

	switch tok.Type {
	case TokenLParen:
		p.consume()
		expr, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		if p.peek().Type != TokenRParen {
			return nil, fmt.Errorf("expected ')' at position %d", p.peek().Pos)
		}
		p.consume()
		return expr, nil

	case TokenIdentifier:
		p.consume()
		if !p.fields[strings.ToLower(tok.Value)] {
			return nil, fmt.Errorf("unknown field '%s' at position %d", tok.Value, tok.Pos)
		}
		return &FieldExpr{Name: strings.ToLower(tok.Value)}, nil

	case TokenNumber:
		p.consume()
		return &LiteralExpr{Type: TokenNumber, Value: tok.Value}, nil

	case TokenString:
		p.consume()
		return &LiteralExpr{Type: TokenString, Value: tok.Value}, nil

	case TokenBool:
		p.consume()
		return &LiteralExpr{Type: TokenBool, Value: tok.Value}, nil

	default:
		return nil, fmt.Errorf("unexpected token '%s' at position %d", tok.Value, tok.Pos)
	}
}

func tokenOpStr(op TokenType) string {
	switch op {
	case TokenOpEQ:
		return "=="
	case TokenOpNE:
		return "!="
	case TokenOpLT:
		return "<"
	case TokenOpGT:
		return ">"
	case TokenOpLE:
		return "<="
	case TokenOpGE:
		return ">="
	case TokenOpAnd:
		return "&&"
	case TokenOpOr:
		return "||"
	case TokenOpNot:
		return "!"
	case TokenOpContains:
		return "contains"
	case TokenOpMatches:
		return "matches"
	default:
		return "?"
	}
}

func (p *ExpressionParser) Evaluate(expr ExprNode, result *models.TestResult) bool {
	switch node := expr.(type) {
	case *BinaryExpr:
		return p.evalBinary(node, result)
	case *UnaryExpr:
		return p.evalUnary(node, result)
	case *FieldExpr:
		val := p.getFieldValue(node.Name, result)
		return p.truthy(val)
	case *LiteralExpr:
		return p.truthy(node.Value)
	default:
		return false
	}
}

func (p *ExpressionParser) evalBinary(node *BinaryExpr, result *models.TestResult) bool {
	leftVal := p.getExprValue(node.Left, result)
	rightVal := p.getExprValue(node.Right, result)

	switch node.Op {
	case TokenOpEQ:
		return p.compareEQ(leftVal, rightVal)
	case TokenOpNE:
		return !p.compareEQ(leftVal, rightVal)
	case TokenOpLT:
		return p.compareNum(leftVal, rightVal) < 0
	case TokenOpGT:
		return p.compareNum(leftVal, rightVal) > 0
	case TokenOpLE:
		return p.compareNum(leftVal, rightVal) <= 0
	case TokenOpGE:
		return p.compareNum(leftVal, rightVal) >= 0
	case TokenOpAnd:
		return p.truthy(leftVal) && p.truthy(rightVal)
	case TokenOpOr:
		return p.truthy(leftVal) || p.truthy(rightVal)
	case TokenOpContains:
		return strings.Contains(p.asString(leftVal), p.asString(rightVal))
	case TokenOpMatches:
		re, err := regexp.Compile(p.asString(rightVal))
		if err != nil {
			return false
		}
		return re.MatchString(p.asString(leftVal))
	default:
		return false
	}
}

func (p *ExpressionParser) evalUnary(node *UnaryExpr, result *models.TestResult) bool {
	val := p.Evaluate(node.Operand, result)
	if node.Op == TokenOpNot {
		return !val
	}
	return val
}

func (p *ExpressionParser) getExprValue(expr ExprNode, result *models.TestResult) interface{} {
	switch node := expr.(type) {
	case *FieldExpr:
		return p.getFieldValue(node.Name, result)
	case *LiteralExpr:
		switch node.Type {
		case TokenNumber:
			if f, err := strconv.ParseFloat(node.Value, 64); err == nil {
				return f
			}
			return node.Value
		case TokenBool:
			return node.Value == "true"
		default:
			return node.Value
		}
	default:
		return nil
	}
}

func (p *ExpressionParser) getFieldValue(name string, result *models.TestResult) interface{} {
	switch name {
	case "status":
		if result.Response != nil {
			return float64(result.Response.StatusCode)
		}
		return 0.0
	case "latency":
		if result.Response != nil {
			return float64(result.Response.Latency.Milliseconds())
		}
		return 0.0
	case "duration":
		return float64(result.Duration.Milliseconds())
	case "name":
		return result.CaseName
	case "id":
		return result.CaseID
	case "passed":
		return result.Status == "passed"
	case "failed":
		return result.Status == "failed"
	case "skipped":
		return result.Status == "skipped"
	case "tag":
		return result.CaseName
	default:
		return nil
	}
}

func (p *ExpressionParser) truthy(val interface{}) bool {
	switch v := val.(type) {
	case bool:
		return v
	case string:
		return v != ""
	case float64:
		return v != 0
	case int:
		return v != 0
	case time.Duration:
		return v != 0
	default:
		return val != nil
	}
}

func (p *ExpressionParser) compareEQ(a, b interface{}) bool {
	if a == nil || b == nil {
		return a == b
	}

	af, aok := a.(float64)
	bf, bok := b.(float64)
	if aok && bok {
		return af == bf
	}

	ab, aok := a.(bool)
	bb, bok := b.(bool)
	if aok && bok {
		return ab == bb
	}

	return p.asString(a) == p.asString(b)
}

func (p *ExpressionParser) compareNum(a, b interface{}) int {
	af, aok := a.(float64)
	bf, bok := b.(float64)
	if aok && bok {
		if af < bf {
			return -1
		} else if af > bf {
			return 1
		}
		return 0
	}

	as := p.asString(a)
	bs := p.asString(b)
	if as < bs {
		return -1
	} else if as > bs {
		return 1
	}
	return 0
}

func (p *ExpressionParser) asString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case int:
		return strconv.Itoa(val)
	case bool:
		return strconv.FormatBool(val)
	case time.Duration:
		return strconv.FormatInt(val.Milliseconds(), 10)
	default:
		return fmt.Sprintf("%v", val)
	}
}
