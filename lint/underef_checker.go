package lint

import (
	"go/ast"
	"go/types"

	"github.com/go-critic/go-critic/lint/internal/lintutil"
	"github.com/go-toolsmith/astp"
)

func init() {
	addChecker(&underefChecker{})
}

type underefChecker struct {
	checkerBase

	skipRecvCopy bool
}

func (c *underefChecker) InitDocumentation(d *Documentation) {
	d.Summary = "Detects dereference expressions that can be omitted"
	d.Before = `
(*k).field = 5
v := (*a)[5] // only if a is array`
	d.After = `
k.field = 5
v := a[5]`
}

func (c *underefChecker) Init() {
	c.skipRecvCopy = c.ctx.params.Bool("skipRecvCopy", true)
}

func (c *underefChecker) VisitExpr(expr ast.Expr) {
	switch n := expr.(type) {
	case *ast.SelectorExpr:
		expr := lintutil.AsParenExpr(n.X)
		if c.skipRecvCopy && c.isPtrRecvMethodCall(n.Sel) {
			return
		}

		if expr, ok := expr.X.(*ast.StarExpr); ok {
			if c.checkStarExpr(expr) {
				c.warnSelect(n)
			}
		}
	case *ast.IndexExpr:
		expr := lintutil.AsParenExpr(n.X)
		if expr, ok := expr.X.(*ast.StarExpr); ok {
			if !c.checkStarExpr(expr) {
				return
			}
			if c.checkArray(expr) {
				c.warnArray(n)
			}
		}
	}
}

func (c *underefChecker) isPtrRecvMethodCall(fn *ast.Ident) bool {
	typ, ok := c.ctx.typesInfo.TypeOf(fn).(*types.Signature)
	if ok && typ != nil && typ.Recv() != nil {
		_, ok := typ.Recv().Type().(*types.Pointer)
		return ok && c.skipRecvCopy
	}
	return false
}

func (c *underefChecker) underef(x *ast.ParenExpr) ast.Expr {
	// If there is only 1 deref, can remove parenthesis,
	// otherwise can remove StarExpr only.
	dereferenced := x.X.(*ast.StarExpr).X
	if astp.IsStarExpr(dereferenced) {
		return &ast.ParenExpr{X: dereferenced}
	}
	return dereferenced
}

func (c *underefChecker) warnSelect(expr *ast.SelectorExpr) {
	// TODO: add () to function output.
	c.ctx.Warn(expr, "could simplify %s to %s.%s",
		expr,
		c.underef(expr.X.(*ast.ParenExpr)),
		expr.Sel.Name)
}

func (c *underefChecker) warnArray(expr *ast.IndexExpr) {
	c.ctx.Warn(expr, "could simplify %s to %s[%s]",
		expr,
		c.underef(expr.X.(*ast.ParenExpr)),
		expr.Index)
}

// checkStarExpr checks if ast.StarExpr could be simplified.
func (c *underefChecker) checkStarExpr(expr *ast.StarExpr) bool {
	typ, ok := c.ctx.typesInfo.TypeOf(expr.X).Underlying().(*types.Pointer)
	if !ok {
		return false
	}

	switch typ.Elem().Underlying().(type) {
	case *types.Pointer, *types.Interface:
		return false
	default:
		return true
	}
}

func (c *underefChecker) checkArray(expr *ast.StarExpr) bool {
	typ, ok := c.ctx.typesInfo.TypeOf(expr.X).(*types.Pointer)
	if !ok {
		return false
	}
	_, ok = typ.Elem().(*types.Array)
	return ok
}
