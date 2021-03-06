package astwalk

import "go/ast"

type localCommentWalker struct {
	visitor LocalCommentVisitor
}

func (w *localCommentWalker) WalkFile(f *ast.File) {
	for _, decl := range f.Decls {
		decl, ok := decl.(*ast.FuncDecl)
		if !ok || !w.visitor.EnterFunc(decl) {
			continue
		}
		for _, c := range f.Comments {
			// Not sure that decls/comments are sorted
			// by positions, so do a naive full scan for now.
			if c.Pos() < decl.Pos() || c.Pos() > decl.End() {
				continue
			}
			w.visitor.VisitLocalComment(c)
		}
	}
}
