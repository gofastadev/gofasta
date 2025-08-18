package transpiler

import (
	"fmt"
	"go/token"
)

// TokenType represents the type of tokens
type TokenType int

const (
	// Special tokens
	ILLEGAL TokenType = iota
	EOF
	COMMENT

	// Literals
	IDENT   // identifiers
	INT     // integers
	FLOAT   // floats
	STRING  // string literals
	CHAR    // character literals
	BOOLEAN // true/false

	// Operators
	ASSIGN   // =
	PLUS     // +
	MINUS    // -
	MULTIPLY // *
	DIVIDE   // /
	MODULO   // %

	// Comparison
	EQ     // ==
	NOT_EQ // !=
	LT     // <
	GT     // >
	LTE    // <=
	GTE    // >=

	// Logical
	AND // &&
	OR  // ||
	NOT // !

	// Punctuation
	SEMICOLON // ;
	COMMA     // ,
	PERIOD    // .
	COLON     // :

	// Delimiters
	LPAREN   // (
	RPAREN   // )
	LBRACE   // {
	RBRACE   // }
	LBRACKET // [
	RBRACKET // ]

	// Keywords
	PACKAGE   // package
	IMPORT    // import
	TYPE      // type
	STRUCT    // struct
	INTERFACE // interface
	FUNC      // func
	VAR       // var
	CONST     // const
	IF        // if
	ELSE      // else
	FOR       // for
	RETURN    // return

	// Gofasta-specific tokens
	DECORATOR // @
	ARROW     // =>
	SPREAD    // ...
	QUESTION  // ?

	// Go types
	GO_INT     // int
	GO_STRING  // string
	GO_BOOL    // bool
	GO_FLOAT   // float64
	GO_SLICE   // []
	GO_MAP     // map
	GO_CHAN    // chan
	GO_POINTER // *
	GO_ERROR   // error
)

// Token represents a single token
type Token struct {
	Type     TokenType
	Literal  string
	Position token.Pos
	Line     int
	Column   int
}

// String returns a string representation of the token
func (t Token) String() string {
	return fmt.Sprintf("Token{Type: %s, Literal: %q, Line: %d, Column: %d}",
		tokenTypeNames[t.Type], t.Literal, t.Line, t.Column)
}

// tokenTypeNames maps TokenType to readable names
var tokenTypeNames = map[TokenType]string{
	ILLEGAL:    "ILLEGAL",
	EOF:        "EOF",
	COMMENT:    "COMMENT",
	IDENT:      "IDENT",
	INT:        "INT",
	FLOAT:      "FLOAT",
	STRING:     "STRING",
	CHAR:       "CHAR",
	BOOLEAN:    "BOOLEAN",
	ASSIGN:     "ASSIGN",
	PLUS:       "PLUS",
	MINUS:      "MINUS",
	MULTIPLY:   "MULTIPLY",
	DIVIDE:     "DIVIDE",
	MODULO:     "MODULO",
	EQ:         "EQ",
	NOT_EQ:     "NOT_EQ",
	LT:         "LT",
	GT:         "GT",
	LTE:        "LTE",
	GTE:        "GTE",
	AND:        "AND",
	OR:         "OR",
	NOT:        "NOT",
	SEMICOLON:  "SEMICOLON",
	COMMA:      "COMMA",
	PERIOD:     "PERIOD",
	COLON:      "COLON",
	LPAREN:     "LPAREN",
	RPAREN:     "RPAREN",
	LBRACE:     "LBRACE",
	RBRACE:     "RBRACE",
	LBRACKET:   "LBRACKET",
	RBRACKET:   "RBRACKET",
	PACKAGE:    "PACKAGE",
	IMPORT:     "IMPORT",
	TYPE:       "TYPE",
	STRUCT:     "STRUCT",
	INTERFACE:  "INTERFACE",
	FUNC:       "FUNC",
	VAR:        "VAR",
	CONST:      "CONST",
	IF:         "IF",
	ELSE:       "ELSE",
	FOR:        "FOR",
	RETURN:     "RETURN",
	DECORATOR:  "DECORATOR",
	ARROW:      "ARROW",
	SPREAD:     "SPREAD",
	QUESTION:   "QUESTION",
	GO_INT:     "GO_INT",
	GO_STRING:  "GO_STRING",
	GO_BOOL:    "GO_BOOL",
	GO_FLOAT:   "GO_FLOAT",
	GO_SLICE:   "GO_SLICE",
	GO_MAP:     "GO_MAP",
	GO_CHAN:    "GO_CHAN",
	GO_POINTER: "GO_POINTER",
	GO_ERROR:   "GO_ERROR",
}

// Keywords map
var keywords = map[string]TokenType{
	"package":   PACKAGE,
	"import":    IMPORT,
	"type":      TYPE,
	"struct":    STRUCT,
	"interface": INTERFACE,
	"func":      FUNC,
	"var":       VAR,
	"const":     CONST,
	"if":        IF,
	"else":      ELSE,
	"for":       FOR,
	"return":    RETURN,
	"true":      BOOLEAN,
	"false":     BOOLEAN,
	"int":       GO_INT,
	"string":    GO_STRING,
	"bool":      GO_BOOL,
	"float64":   GO_FLOAT,
	"error":     GO_ERROR,
	"map":       GO_MAP,
	"chan":      GO_CHAN,
}

// Lexer represents the lexer state
type Lexer struct {
	input        string
	position     int  // current position in input (points to current char)
	readPosition int  // current reading position in input (after current char)
	ch           byte // current char under examination
	line         int  // current line number
	column       int  // current column number
}

// NewLexer creates a new lexer instance
func NewLexer(input string) *Lexer {
	l := &Lexer{
		input:  input,
		line:   1,
		column: 0,
	}
	l.readChar()
	return l
}

// readChar gives us the next character and advances our position in the input string
func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0 // ASCII "NUL" character represents "haven't read anything yet" or "EOF"
	} else {
		l.ch = l.input[l.readPosition]
	}
	l.position = l.readPosition
	l.readPosition++

	if l.ch == '\n' {
		l.line++
		l.column = 0
	} else {
		l.column++
	}
}

// peekChar returns the next character without advancing our position
func (l *Lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}

// NextToken scans the input and returns the next token
func (l *Lexer) NextToken() Token {
	var tok Token

	l.skipWhitespace()

	pos := token.Pos(l.position)
	line := l.line
	column := l.column

	switch l.ch {
	case '@':
		tok = Token{Type: DECORATOR, Literal: string(l.ch), Position: pos, Line: line, Column: column}
	case '=':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: EQ, Literal: string(ch) + string(l.ch), Position: pos, Line: line, Column: column}
		} else if l.peekChar() == '>' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: ARROW, Literal: string(ch) + string(l.ch), Position: pos, Line: line, Column: column}
		} else {
			tok = Token{Type: ASSIGN, Literal: string(l.ch), Position: pos, Line: line, Column: column}
		}
	case '!':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: NOT_EQ, Literal: string(ch) + string(l.ch), Position: pos, Line: line, Column: column}
		} else {
			tok = Token{Type: NOT, Literal: string(l.ch), Position: pos, Line: line, Column: column}
		}
	case '<':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: LTE, Literal: string(ch) + string(l.ch), Position: pos, Line: line, Column: column}
		} else {
			tok = Token{Type: LT, Literal: string(l.ch), Position: pos, Line: line, Column: column}
		}
	case '>':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: GTE, Literal: string(ch) + string(l.ch), Position: pos, Line: line, Column: column}
		} else {
			tok = Token{Type: GT, Literal: string(l.ch), Position: pos, Line: line, Column: column}
		}
	case '&':
		if l.peekChar() == '&' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: AND, Literal: string(ch) + string(l.ch), Position: pos, Line: line, Column: column}
		} else {
			tok = Token{Type: ILLEGAL, Literal: string(l.ch), Position: pos, Line: line, Column: column}
		}
	case '|':
		if l.peekChar() == '|' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: OR, Literal: string(ch) + string(l.ch), Position: pos, Line: line, Column: column}
		} else {
			tok = Token{Type: ILLEGAL, Literal: string(l.ch), Position: pos, Line: line, Column: column}
		}
	case '+':
		tok = Token{Type: PLUS, Literal: string(l.ch), Position: pos, Line: line, Column: column}
	case '-':
		tok = Token{Type: MINUS, Literal: string(l.ch), Position: pos, Line: line, Column: column}
	case '*':
		tok = Token{Type: MULTIPLY, Literal: string(l.ch), Position: pos, Line: line, Column: column}
	case '/':
		if l.peekChar() == '/' {
			return l.readSingleLineComment()
		} else if l.peekChar() == '*' {
			return l.readMultiLineComment()
		}
		tok = Token{Type: DIVIDE, Literal: string(l.ch), Position: pos, Line: line, Column: column}
	case '%':
		tok = Token{Type: MODULO, Literal: string(l.ch), Position: pos, Line: line, Column: column}
	case ';':
		tok = Token{Type: SEMICOLON, Literal: string(l.ch), Position: pos, Line: line, Column: column}
	case ',':
		tok = Token{Type: COMMA, Literal: string(l.ch), Position: pos, Line: line, Column: column}
	case '.':
		if l.peekChar() == '.' && l.peekCharAt(2) == '.' {
			l.readChar()
			l.readChar()
			tok = Token{Type: SPREAD, Literal: "...", Position: pos, Line: line, Column: column}
		} else {
			tok = Token{Type: PERIOD, Literal: string(l.ch), Position: pos, Line: line, Column: column}
		}
	case ':':
		tok = Token{Type: COLON, Literal: string(l.ch), Position: pos, Line: line, Column: column}
	case '?':
		tok = Token{Type: QUESTION, Literal: string(l.ch), Position: pos, Line: line, Column: column}
	case '(':
		tok = Token{Type: LPAREN, Literal: string(l.ch), Position: pos, Line: line, Column: column}
	case ')':
		tok = Token{Type: RPAREN, Literal: string(l.ch), Position: pos, Line: line, Column: column}
	case '{':
		tok = Token{Type: LBRACE, Literal: string(l.ch), Position: pos, Line: line, Column: column}
	case '}':
		tok = Token{Type: RBRACE, Literal: string(l.ch), Position: pos, Line: line, Column: column}
	case '[':
		tok = Token{Type: LBRACKET, Literal: string(l.ch), Position: pos, Line: line, Column: column}
	case ']':
		tok = Token{Type: RBRACKET, Literal: string(l.ch), Position: pos, Line: line, Column: column}
	case '"':
		tok.Type = STRING
		tok.Literal = l.readString()
		tok.Position = pos
		tok.Line = line
		tok.Column = column
	case '`':
		tok.Type = STRING
		tok.Literal = l.readRawString()
		tok.Position = pos
		tok.Line = line
		tok.Column = column
	case '\'':
		tok.Type = CHAR
		tok.Literal = l.readChar2()
		tok.Position = pos
		tok.Line = line
		tok.Column = column
	case 0:
		tok = Token{Type: EOF, Literal: "", Position: pos, Line: line, Column: column}
	default:
		if isLetter(l.ch) {
			tok.Literal = l.readIdentifier()
			tok.Type = lookupIdent(tok.Literal)
			tok.Position = pos
			tok.Line = line
			tok.Column = column
			return tok // early return to avoid calling readChar
		} else if isDigit(l.ch) {
			tok.Type, tok.Literal = l.readNumber()
			tok.Position = pos
			tok.Line = line
			tok.Column = column
			return tok // early return to avoid calling readChar
		} else {
			tok = Token{Type: ILLEGAL, Literal: string(l.ch), Position: pos, Line: line, Column: column}
		}
	}

	l.readChar()
	return tok
}

// skipWhitespace skips whitespace characters
func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		l.readChar()
	}
}

// readIdentifier reads an identifier
func (l *Lexer) readIdentifier() string {
	position := l.position
	for isLetter(l.ch) || isDigit(l.ch) {
		l.readChar()
	}
	return l.input[position:l.position]
}

// readNumber reads a number (integer or float)
func (l *Lexer) readNumber() (TokenType, string) {
	position := l.position
	tokenType := INT

	for isDigit(l.ch) {
		l.readChar()
	}

	// Check for decimal point
	if l.ch == '.' && isDigit(l.peekChar()) {
		tokenType = FLOAT
		l.readChar() // consume '.'
		for isDigit(l.ch) {
			l.readChar()
		}
	}

	return tokenType, l.input[position:l.position]
}

// readString reads a string literal
func (l *Lexer) readString() string {
	position := l.position + 1 // skip opening quote
	for {
		l.readChar()
		if l.ch == '"' || l.ch == 0 {
			break
		}
		// Handle escape sequences
		if l.ch == '\\' {
			l.readChar()
		}
	}
	return l.input[position:l.position]
}

// readRawString reads a raw string literal (backticks)
func (l *Lexer) readRawString() string {
	position := l.position + 1 // skip opening backtick
	for {
		l.readChar()
		if l.ch == '`' || l.ch == 0 {
			break
		}
	}
	return l.input[position:l.position]
}

// readChar2 reads a character literal
func (l *Lexer) readChar2() string {
	position := l.position + 1 // skip opening quote
	for {
		l.readChar()
		if l.ch == '\'' || l.ch == 0 {
			break
		}
		// Handle escape sequences
		if l.ch == '\\' {
			l.readChar()
		}
	}
	return l.input[position:l.position]
}

// readSingleLineComment reads a single-line comment
func (l *Lexer) readSingleLineComment() Token {
	position := l.position
	line := l.line
	column := l.column

	for l.ch != '\n' && l.ch != 0 {
		l.readChar()
	}

	return Token{
		Type:     COMMENT,
		Literal:  l.input[position:l.position],
		Position: token.Pos(position),
		Line:     line,
		Column:   column,
	}
}

// readMultiLineComment reads a multi-line comment
func (l *Lexer) readMultiLineComment() Token {
	position := l.position
	line := l.line
	column := l.column

	l.readChar() // skip '/'
	l.readChar() // skip '*'

	for {
		if l.ch == 0 {
			break
		}
		if l.ch == '*' && l.peekChar() == '/' {
			l.readChar() // skip '*'
			l.readChar() // skip '/'
			break
		}
		l.readChar()
	}

	return Token{
		Type:     COMMENT,
		Literal:  l.input[position:l.position],
		Position: token.Pos(position),
		Line:     line,
		Column:   column,
	}
}

// peekCharAt returns the character at the given offset without advancing position
func (l *Lexer) peekCharAt(offset int) byte {
	pos := l.readPosition + offset - 1
	if pos >= len(l.input) {
		return 0
	}
	return l.input[pos]
}

// isLetter checks if a character is a letter
func isLetter(ch byte) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_'
}

// isDigit checks if a character is a digit
func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}

// lookupIdent determines if an identifier is a keyword
func lookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}

// TokenizeFile tokenizes an entire .gofa file
func TokenizeFile(input string) ([]Token, error) {
	lexer := NewLexer(input)
	var tokens []Token

	for {
		token := lexer.NextToken()
		tokens = append(tokens, token)

		if token.Type == EOF {
			break
		}

		if token.Type == ILLEGAL {
			return nil, fmt.Errorf("illegal token %q at line %d, column %d",
				token.Literal, token.Line, token.Column)
		}
	}

	return tokens, nil
}

// IsValidGoFastaFile checks if the file contains Gofasta decorators
func IsValidGoFastaFile(tokens []Token) bool {
	for _, token := range tokens {
		if token.Type == DECORATOR {
			return true
		}
	}
	return false
}

// FilterTokens filters tokens by type
func FilterTokens(tokens []Token, tokenType TokenType) []Token {
	var filtered []Token
	for _, token := range tokens {
		if token.Type == tokenType {
			filtered = append(filtered, token)
		}
	}
	return filtered
}

// FindDecoratorTokens finds all decorator tokens in a token stream
func FindDecoratorTokens(tokens []Token) []Token {
	return FilterTokens(tokens, DECORATOR)
}
