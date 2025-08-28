package core

import (
	"testing"
)

// testVisitor implements the Visitor interface for testing
type testVisitor struct {
	visitedNodes []string
}

// Visit implements the Visitor interface
func (v *testVisitor) Visit(node GofaASTNode) Visitor {
	if node == nil {
		return nil
	}
	
	switch n := node.(type) {
	case *GofaFile:
		v.visitedNodes = append(v.visitedNodes, "GofaFile")
	case *ControllerDeclaration:
		v.visitedNodes = append(v.visitedNodes, "Controller:"+n.Name)
	case *ServiceDeclaration:
		v.visitedNodes = append(v.visitedNodes, "Service:"+n.Name)
	case *DecoratorNode:
		v.visitedNodes = append(v.visitedNodes, "Decorator:"+n.Name)
	case *FieldNode:
		v.visitedNodes = append(v.visitedNodes, "Field:"+n.Name)
	case *MethodNode:
		v.visitedNodes = append(v.visitedNodes, "Method:"+n.Name)
	case *ParameterNode:
		v.visitedNodes = append(v.visitedNodes, "Param:"+n.Name)
	}
	return v
}

// countingVisitor counts nodes visited
type countingVisitor struct {
	nodeCount int
}

func (v *countingVisitor) Visit(node GofaASTNode) Visitor {
	v.nodeCount++
	return v
}

// Test AST Walk function 
func TestWalkFunction(t *testing.T) {
	// Test ServiceDeclaration
	serviceDecl := &ServiceDeclaration{
		Name: "UserService",
		Decorators: []*DecoratorNode{
			{Name: "Injectable"},
		},
	}
	
	visitor := &testVisitor{visitedNodes: []string{}}
	Walk(visitor, serviceDecl)
	
	// Verify that we visited some nodes
	if len(visitor.visitedNodes) == 0 {
		t.Error("Expected to visit some nodes, but visited none")
	}
	
	t.Logf("Visited nodes for ServiceDeclaration: %v", visitor.visitedNodes)
}

// Test Walk with various declaration types
func TestWalkDifferentDeclarations(t *testing.T) {
	// Test ModuleDeclaration
	moduleDecl := &ModuleDeclaration{
		Name: "AppModule",
		Decorators: []*DecoratorNode{
			{Name: "Module"},
		},
	}
	
	visitor := &countingVisitor{}
	Walk(visitor, moduleDecl)
	
	if visitor.nodeCount == 0 {
		t.Error("Expected to visit some nodes for ModuleDeclaration")
	}
	
	t.Logf("Visited %d nodes for ModuleDeclaration", visitor.nodeCount)
	
	// Test WebSocketGatewayDeclaration
	wsDecl := &WebSocketGatewayDeclaration{
		Name: "ChatGateway",
		Decorators: []*DecoratorNode{
			{Name: "WebSocketGateway"},
		},
		Fields: []*FieldNode{
			{Name: "server", Type: "*ws.Server"},
		},
	}
	
	visitor2 := &countingVisitor{}
	Walk(visitor2, wsDecl)
	
	if visitor2.nodeCount == 0 {
		t.Error("Expected to visit some nodes for WebSocketGatewayDeclaration")
	}
	
	t.Logf("Visited %d nodes for WebSocketGatewayDeclaration", visitor2.nodeCount)
}

// Test Walk with TestSuiteDeclaration (uncovered case)
func TestWalkTestSuite(t *testing.T) {
	testSuiteDecl := &TestSuiteDeclaration{
		Name: "UserServiceTest",
		Decorators: []*DecoratorNode{
			{Name: "TestSuite"},
		},
		Fields: []*FieldNode{
			{Name: "userService", Type: "*MockUserService"},
		},
		Methods: []*MethodNode{
			{
				Name: "TestCreateUser",
				Params: []*ParameterNode{
					{Name: "t", Type: "*testing.T"},
				},
			},
		},
	}
	
	visitor := &countingVisitor{}
	Walk(visitor, testSuiteDecl)
	
	if visitor.nodeCount == 0 {
		t.Error("Expected to visit some nodes for TestSuiteDeclaration")
	}
	
	t.Logf("Visited %d nodes for TestSuiteDeclaration", visitor.nodeCount)
}