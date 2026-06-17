package variables

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"strconv"
	"strings"
)

var (
	varRegex       = regexp.MustCompile(`\{\{\s*([^}]+)\s*\}\}`)
	funcRegex      = regexp.MustCompile(`^(\$\w+)\((.*)\)$`)
	mathExprRegex  = regexp.MustCompile(`^[\d\s+\-*/().]+$`)
)

type Interpolator struct {
	store *VariableStore
}

func NewInterpolator(store *VariableStore) *Interpolator {
	return &Interpolator{
		store: store,
	}
}

func (ip *Interpolator) InterpolateString(s string) (string, error) {
	if !varRegex.MatchString(s) {
		return s, nil
	}

	result := s
	var err error
	for i := 0; i < 10; i++ {
		replaced := false
		result = varRegex.ReplaceAllStringFunc(result, func(match string) string {
			inner := strings.TrimSpace(match[2 : len(match)-2])
			value, e := ip.evaluateExpression(inner)
			if e != nil {
				err = e
				return match
			}
			replaced = true
			return fmt.Sprintf("%v", value)
		})
		if err != nil {
			return "", err
		}
		if !replaced || !varRegex.MatchString(result) {
			break
		}
	}

	return result, nil
}

func (ip *Interpolator) InterpolateMap(m map[string]any) (map[string]any, error) {
	result := make(map[string]any)
	for k, v := range m {
		newV, err := ip.InterpolateInterface(v)
		if err != nil {
			return nil, err
		}
		result[k] = newV
	}
	return result, nil
}

func (ip *Interpolator) InterpolateInterface(v any) (any, error) {
	switch val := v.(type) {
	case string:
		return ip.InterpolateString(val)
	case map[string]any:
		return ip.InterpolateMap(val)
	case map[any]any:
		m := make(map[string]any)
		for k, vv := range val {
			keyStr, ok := k.(string)
			if !ok {
				return nil, fmt.Errorf("non-string key in map: %v", k)
			}
			newV, err := ip.InterpolateInterface(vv)
			if err != nil {
				return nil, err
			}
			m[keyStr] = newV
		}
		return m, nil
	case []any:
		arr := make([]any, len(val))
		for i, item := range val {
			newItem, err := ip.InterpolateInterface(item)
			if err != nil {
				return nil, err
			}
			arr[i] = newItem
		}
		return arr, nil
	case []string:
		arr := make([]string, len(val))
		for i, item := range val {
			newItem, err := ip.InterpolateString(item)
			if err != nil {
				return nil, err
			}
			arr[i] = newItem
		}
		return arr, nil
	case map[string]string:
		m := make(map[string]string)
		for k, vv := range val {
			newV, err := ip.InterpolateString(vv)
			if err != nil {
				return nil, err
			}
			m[k] = newV
		}
		return m, nil
	default:
		return v, nil
	}
}

func (ip *Interpolator) evaluateExpression(expr string) (any, error) {
	expr = strings.TrimSpace(expr)

	if funcMatch := funcRegex.FindStringSubmatch(expr); funcMatch != nil {
		funcName := funcMatch[1]
		argsStr := funcMatch[2]
		args := parseFunctionArgs(argsStr)
		return CallFunction(funcName, args)
	}

	if strings.HasPrefix(expr, "$") {
		if v, ok := ip.store.Get(expr); ok {
			return v, nil
		}
		if strings.HasPrefix(expr, "$env.") {
			envName := strings.TrimPrefix(expr, "$env.")
			if v, ok := ip.store.Get("$env." + envName); ok {
				return v, nil
			}
		}
		return "", fmt.Errorf("variable not found: %s", expr)
	}

	if mathExprRegex.MatchString(expr) && containsOperator(expr) {
		return evaluateMathExpression(expr)
	}

	if v, ok := ip.store.Get(expr); ok {
		return v, nil
	}

	if num, err := strconv.ParseFloat(expr, 64); err == nil {
		return num, nil
	}

	if strings.HasPrefix(expr, "'") && strings.HasSuffix(expr, "'") {
		return expr[1 : len(expr)-1], nil
	}
	if strings.HasPrefix(expr, "\"") && strings.HasSuffix(expr, "\"") {
		return expr[1 : len(expr)-1], nil
	}

	return expr, nil
}

func parseFunctionArgs(argsStr string) []string {
	if strings.TrimSpace(argsStr) == "" {
		return []string{}
	}

	var args []string
	var current strings.Builder
	depth := 0
	inQuote := false
	quoteChar := rune(0)

	for _, r := range argsStr {
		switch {
		case (r == '\'' || r == '"') && !inQuote:
			inQuote = true
			quoteChar = r
			current.WriteRune(r)
		case r == quoteChar && inQuote:
			inQuote = false
			quoteChar = rune(0)
			current.WriteRune(r)
		case r == '(' && !inQuote:
			depth++
			current.WriteRune(r)
		case r == ')' && !inQuote:
			depth--
			current.WriteRune(r)
		case r == ',' && !inQuote && depth == 0:
			args = append(args, current.String())
			current.Reset()
		default:
			current.WriteRune(r)
		}
	}

	if current.Len() > 0 {
		args = append(args, current.String())
	}

	return args
}

func containsOperator(expr string) bool {
	for _, op := range []rune{'+', '-', '*', '/'} {
		if strings.ContainsRune(expr, op) {
			return true
		}
	}
	return false
}

func evaluateMathExpression(expr string) (float64, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return 0, fmt.Errorf("empty expression")
	}

	tree, err := parser.ParseExpr(expr)
	if err != nil {
		return 0, fmt.Errorf("parse error: %w", err)
	}

	return evalNode(tree)
}

func evalNode(node ast.Expr) (float64, error) {
	switch n := node.(type) {
	case *ast.BasicLit:
		if n.Kind == token.INT || n.Kind == token.FLOAT {
			return strconv.ParseFloat(n.Value, 64)
		}
		return 0, fmt.Errorf("unsupported literal type: %v", n.Kind)
	case *ast.BinaryExpr:
		left, err := evalNode(n.X)
		if err != nil {
			return 0, err
		}
		right, err := evalNode(n.Y)
		if err != nil {
			return 0, err
		}
		switch n.Op {
		case token.ADD:
			return left + right, nil
		case token.SUB:
			return left - right, nil
		case token.MUL:
			return left * right, nil
		case token.QUO:
			if right == 0 {
				return 0, fmt.Errorf("division by zero")
			}
			return left / right, nil
		default:
			return 0, fmt.Errorf("unsupported operator: %v", n.Op)
		}
	case *ast.ParenExpr:
		return evalNode(n.X)
	case *ast.UnaryExpr:
		val, err := evalNode(n.X)
		if err != nil {
			return 0, err
		}
		switch n.Op {
		case token.SUB:
			return -val, nil
		case token.ADD:
			return val, nil
		default:
			return 0, fmt.Errorf("unsupported unary operator: %v", n.Op)
		}
	default:
		return 0, fmt.Errorf("unsupported expression type: %T", node)
	}
}
