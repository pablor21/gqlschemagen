package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ============================================================================
// Field Filtering Tests
// ============================================================================

func TestFieldIncludeList(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.go")
	testContent := `package test

// @gqlType(name:"User")
// @gqlType(name:"Admin")
type Person struct {
	ID       string ` + "`gql:\"id,type:ID\"`" + `
	Name     string
	Password string ` + "`gql:\"password,include:Admin\"`" + `
}
`
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	parser := NewParser()
	if err := parser.Walk(tmpDir); err != nil {
		t.Fatalf("Parser walk failed: %v", err)
	}

	outFile := filepath.Join(tmpDir, "schema.graphqls")
	cfg := &Config{
		Packages:            []string{tmpDir},
		Output:              outFile,
		GenStrategy:         GenStrategySingle,
		UseGqlGenDirectives: true,
	}

	engine := NewGenerator(parser, cfg)
	if err := engine.Run(); err != nil {
		t.Fatalf("Generator run failed: %v", err)
	}

	content, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	schema := string(content)

	// User type should NOT have password field
	if strings.Contains(schema, "type User") {
		userStart := strings.Index(schema, "type User")
		userEnd := strings.Index(schema[userStart:], "}")
		userType := schema[userStart : userStart+userEnd]

		if strings.Contains(userType, "password") {
			t.Error("User type should not contain password field (only included in Admin)")
		}
	} else {
		t.Error("User type not found in schema")
	}

	// Admin type should have password field
	if strings.Contains(schema, "type Admin") {
		adminStart := strings.Index(schema, "type Admin")
		adminEnd := strings.Index(schema[adminStart:], "}")
		adminType := schema[adminStart : adminStart+adminEnd]

		if !strings.Contains(adminType, "password") {
			t.Error("Admin type should contain password field")
		}
	} else {
		t.Error("Admin type not found in schema")
	}
}

func TestFieldOmitList(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.go")
	testContent := `package test

// @gqlType(name:"PublicUser")
// @gqlType(name:"PrivateUser")
type User struct {
	ID       string ` + "`gql:\"id,type:ID\"`" + `
	Name     string
	Email    string ` + "`gql:\"email,omit:PublicUser\"`" + `
}
`
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	parser := NewParser()
	if err := parser.Walk(tmpDir); err != nil {
		t.Fatalf("Parser walk failed: %v", err)
	}

	outFile := filepath.Join(tmpDir, "schema.graphqls")
	cfg := &Config{
		Packages:            []string{tmpDir},
		Output:              outFile,
		GenStrategy:         GenStrategySingle,
		UseGqlGenDirectives: true,
	}

	engine := NewGenerator(parser, cfg)
	if err := engine.Run(); err != nil {
		t.Fatalf("Generator run failed: %v", err)
	}

	content, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	schema := string(content)

	// PublicUser type should NOT have email field
	if strings.Contains(schema, "type PublicUser") {
		publicStart := strings.Index(schema, "type PublicUser")
		publicEnd := strings.Index(schema[publicStart:], "}")
		publicType := schema[publicStart : publicStart+publicEnd]

		if strings.Contains(publicType, "email") {
			t.Error("PublicUser type should not contain email field (omitted)")
		}
	} else {
		t.Error("PublicUser type not found in schema")
	}

	// PrivateUser type should have email field
	if strings.Contains(schema, "type PrivateUser") {
		privateStart := strings.Index(schema, "type PrivateUser")
		privateEnd := strings.Index(schema[privateStart:], "}")
		privateType := schema[privateStart : privateStart+privateEnd]

		if !strings.Contains(privateType, "email") {
			t.Error("PrivateUser type should contain email field")
		}
	} else {
		t.Error("PrivateUser type not found in schema")
	}
}

func TestFieldIgnoreList(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.go")
	testContent := `package test

// @gqlType(name:"UserDTO")
// @gqlType(name:"UserEntity")
type User struct {
	ID       string ` + "`gql:\"id,type:ID\"`" + `
	Name     string
	Internal string ` + "`gql:\"internal,ignore:UserDTO\"`" + `
}
`
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	parser := NewParser()
	if err := parser.Walk(tmpDir); err != nil {
		t.Fatalf("Parser walk failed: %v", err)
	}

	outFile := filepath.Join(tmpDir, "schema.graphqls")
	cfg := &Config{
		Packages:            []string{tmpDir},
		Output:              outFile,
		GenStrategy:         GenStrategySingle,
		UseGqlGenDirectives: true,
	}

	engine := NewGenerator(parser, cfg)
	if err := engine.Run(); err != nil {
		t.Fatalf("Generator run failed: %v", err)
	}

	content, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	schema := string(content)

	// UserDTO type should NOT have internal field
	if strings.Contains(schema, "type UserDTO") {
		dtoStart := strings.Index(schema, "type UserDTO")
		dtoEnd := strings.Index(schema[dtoStart:], "}")
		dtoType := schema[dtoStart : dtoStart+dtoEnd]

		if strings.Contains(dtoType, "internal") {
			t.Error("UserDTO type should not contain internal field (ignored)")
		}
	} else {
		t.Error("UserDTO type not found in schema")
	}

	// UserEntity type should have internal field
	if strings.Contains(schema, "type UserEntity") {
		entityStart := strings.Index(schema, "type UserEntity")
		entityEnd := strings.Index(schema[entityStart:], "}")
		entityType := schema[entityStart : entityStart+entityEnd]

		if !strings.Contains(entityType, "internal") {
			t.Error("UserEntity type should contain internal field")
		}
	} else {
		t.Error("UserEntity type not found in schema")
	}
}

func TestFieldReadOnly(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.go")
	testContent := `package test

// @gqlType(name:"User")
// @gqlInput(name:"UserInput")
type UserModel struct {
	ID        string ` + "`gql:\"id,type:ID,ro\"`" + `
	Name      string
	CreatedAt string ` + "`gql:\"createdAt,ro\"`" + `
}
`
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	parser := NewParser()
	if err := parser.Walk(tmpDir); err != nil {
		t.Fatalf("Parser walk failed: %v", err)
	}

	outFile := filepath.Join(tmpDir, "schema.graphqls")
	cfg := &Config{
		Packages:            []string{tmpDir},
		Output:              outFile,
		GenStrategy:         GenStrategySingle,
		UseGqlGenDirectives: true,
	}

	engine := NewGenerator(parser, cfg)
	if err := engine.Run(); err != nil {
		t.Fatalf("Generator run failed: %v", err)
	}

	content, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	schema := string(content)

	// User type should have id and createdAt
	if strings.Contains(schema, "type User") {
		userStart := strings.Index(schema, "type User")
		userEnd := strings.Index(schema[userStart:], "}")
		userType := schema[userStart : userStart+userEnd]

		if !strings.Contains(userType, "id") {
			t.Error("User type should contain id field (read-only)")
		}
		if !strings.Contains(userType, "createdAt") {
			t.Error("User type should contain createdAt field (read-only)")
		}
	} else {
		t.Error("User type not found in schema")
	}

	// UserInput should NOT have id or createdAt (read-only fields)
	if strings.Contains(schema, "input UserInput") {
		inputStart := strings.Index(schema, "input UserInput")
		inputEnd := strings.Index(schema[inputStart:], "}")
		inputType := schema[inputStart : inputStart+inputEnd]

		if strings.Contains(inputType, "id") {
			t.Error("UserInput should not contain id field (read-only)")
		}
		if strings.Contains(inputType, "createdAt") {
			t.Error("UserInput should not contain createdAt field (read-only)")
		}
		if !strings.Contains(inputType, "name") {
			t.Error("UserInput should contain name field")
		}
	} else {
		t.Error("UserInput not found in schema")
	}
}

func TestFieldWriteOnly(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.go")
	testContent := `package test

// @gqlType(name:"User")
// @gqlInput(name:"UserInput")
type UserModel struct {
	ID       string ` + "`gql:\"id,type:ID\"`" + `
	Name     string
	Password string ` + "`gql:\"password,wo\"`" + `
}
`
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	parser := NewParser()
	if err := parser.Walk(tmpDir); err != nil {
		t.Fatalf("Parser walk failed: %v", err)
	}

	outFile := filepath.Join(tmpDir, "schema.graphqls")
	cfg := &Config{
		Packages:            []string{tmpDir},
		Output:              outFile,
		GenStrategy:         GenStrategySingle,
		UseGqlGenDirectives: true,
	}

	engine := NewGenerator(parser, cfg)
	if err := engine.Run(); err != nil {
		t.Fatalf("Generator run failed: %v", err)
	}

	content, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	schema := string(content)

	// User type should NOT have password field (write-only)
	if strings.Contains(schema, "type User") {
		userStart := strings.Index(schema, "type User")
		userEnd := strings.Index(schema[userStart:], "}")
		userType := schema[userStart : userStart+userEnd]

		if strings.Contains(userType, "password") {
			t.Error("User type should not contain password field (write-only)")
		}
		if !strings.Contains(userType, "id") {
			t.Error("User type should contain id field")
		}
		if !strings.Contains(userType, "name") {
			t.Error("User type should contain name field")
		}
	} else {
		t.Error("User type not found in schema")
	}

	// UserInput should have password field
	if strings.Contains(schema, "input UserInput") {
		inputStart := strings.Index(schema, "input UserInput")
		inputEnd := strings.Index(schema[inputStart:], "}")
		inputType := schema[inputStart : inputStart+inputEnd]

		if !strings.Contains(inputType, "password") {
			t.Error("UserInput should contain password field (write-only)")
		}
	} else {
		t.Error("UserInput not found in schema")
	}
}

func TestFieldReadWrite(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.go")
	testContent := `package test

// @gqlType(name:"User")
// @gqlInput(name:"UserInput")
// @gqlIgnoreAll
type UserModel struct {
	ID   string ` + "`gql:\"id,type:ID,rw\"`" + `
	Name string ` + "`gql:\"name,rw\"`" + `
}
`
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	parser := NewParser()
	if err := parser.Walk(tmpDir); err != nil {
		t.Fatalf("Parser walk failed: %v", err)
	}

	outFile := filepath.Join(tmpDir, "schema.graphqls")
	cfg := &Config{
		Packages:            []string{tmpDir},
		Output:              outFile,
		GenStrategy:         GenStrategySingle,
		UseGqlGenDirectives: true,
	}

	engine := NewGenerator(parser, cfg)
	if err := engine.Run(); err != nil {
		t.Fatalf("Generator run failed: %v", err)
	}

	content, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	schema := string(content)

	// User type should have both fields despite @gqlIgnoreAll
	if strings.Contains(schema, "type User") {
		userStart := strings.Index(schema, "type User")
		userEnd := strings.Index(schema[userStart:], "}")
		userType := schema[userStart : userStart+userEnd]

		if !strings.Contains(userType, "id") {
			t.Error("User type should contain id field (rw)")
		}
		if !strings.Contains(userType, "name") {
			t.Error("User type should contain name field (rw)")
		}
	} else {
		t.Error("User type not found in schema")
	}

	// UserInput should have both fields despite @gqlIgnoreAll
	if strings.Contains(schema, "input UserInput") {
		inputStart := strings.Index(schema, "input UserInput")
		inputEnd := strings.Index(schema[inputStart:], "}")
		inputType := schema[inputStart : inputStart+inputEnd]

		if !strings.Contains(inputType, "id") {
			t.Error("UserInput should contain id field (rw)")
		}
		if !strings.Contains(inputType, "name") {
			t.Error("UserInput should contain name field (rw)")
		}
	} else {
		t.Error("UserInput not found in schema")
	}
}

func TestFieldMultipleTypesList(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.go")
	testContent := `package test

// @gqlType(name:"UserV1")
// @gqlType(name:"UserV2")
// @gqlType(name:"UserV3")
type User struct {
	ID      string ` + "`gql:\"id,type:ID\"`" + `
	Name    string
	Email   string ` + "`gql:\"email,include:\\\"UserV2,UserV3\\\"\"`" + `
	Profile string ` + "`gql:\"profile,omit:\\\"UserV1,UserV2\\\"\"`" + `
}
`
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	parser := NewParser()
	if err := parser.Walk(tmpDir); err != nil {
		t.Fatalf("Parser walk failed: %v", err)
	}

	outFile := filepath.Join(tmpDir, "schema.graphqls")
	cfg := &Config{
		Packages:            []string{tmpDir},
		Output:              outFile,
		GenStrategy:         GenStrategySingle,
		UseGqlGenDirectives: true,
	}

	engine := NewGenerator(parser, cfg)
	if err := engine.Run(); err != nil {
		t.Fatalf("Generator run failed: %v", err)
	}

	content, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	schema := string(content)

	// UserV1: should have id, name, but NOT email (not in include list) or profile (in omit list)
	if strings.Contains(schema, "type UserV1") {
		v1Start := strings.Index(schema, "type UserV1")
		v1End := strings.Index(schema[v1Start:], "}")
		v1Type := schema[v1Start : v1Start+v1End]

		if strings.Contains(v1Type, "email") {
			t.Error("UserV1 should not have email (not in include list)")
		}
		if strings.Contains(v1Type, "profile") {
			t.Error("UserV1 should not have profile (in omit list)")
		}
	}

	// UserV2: should have id, name, email (in include list), but NOT profile (in omit list)
	if strings.Contains(schema, "type UserV2") {
		v2Start := strings.Index(schema, "type UserV2")
		v2End := strings.Index(schema[v2Start:], "}")
		v2Type := schema[v2Start : v2Start+v2End]

		if !strings.Contains(v2Type, "email") {
			t.Error("UserV2 should have email (in include list)")
		}
		if strings.Contains(v2Type, "profile") {
			t.Error("UserV2 should not have profile (in omit list)")
		}
	}

	// UserV3: should have all fields (email in include list, profile not in omit list)
	if strings.Contains(schema, "type UserV3") {
		v3Start := strings.Index(schema, "type UserV3")
		v3End := strings.Index(schema[v3Start:], "}")
		v3Type := schema[v3Start : v3Start+v3End]

		if !strings.Contains(v3Type, "email") {
			t.Error("UserV3 should have email (in include list)")
		}
		if !strings.Contains(v3Type, "profile") {
			t.Error("UserV3 should have profile (not in omit list)")
		}
	}
}

func TestFieldWildcardInclude(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.go")
	testContent := `package test

// @gqlType(name:"User")
// @gqlIgnoreAll
type UserModel struct {
	ID   string ` + "`gql:\"id,type:ID,include:*\"`" + `
	Name string ` + "`gql:\"name,include\"`" + `
}
`
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	parser := NewParser()
	if err := parser.Walk(tmpDir); err != nil {
		t.Fatalf("Parser walk failed: %v", err)
	}

	outFile := filepath.Join(tmpDir, "schema.graphqls")
	cfg := &Config{
		Packages:            []string{tmpDir},
		Output:              outFile,
		GenStrategy:         GenStrategySingle,
		UseGqlGenDirectives: true,
	}

	engine := NewGenerator(parser, cfg)
	if err := engine.Run(); err != nil {
		t.Fatalf("Generator run failed: %v", err)
	}

	content, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	schema := string(content)

	// User type should have both fields despite @gqlIgnoreAll (include:* and include)
	if strings.Contains(schema, "type User") {
		userStart := strings.Index(schema, "type User")
		userEnd := strings.Index(schema[userStart:], "}")
		userType := schema[userStart : userStart+userEnd]

		if !strings.Contains(userType, "id") {
			t.Error("User type should contain id field (include:*)")
		}
		if !strings.Contains(userType, "name") {
			t.Error("User type should contain name field (include)")
		}
	} else {
		t.Error("User type not found in schema")
	}
}

func TestFieldReadOnlyWithTypeList(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.go")
	testContent := `package test

// @gqlType(name:"Admin")
// @gqlType(name:"User")
// @gqlInput(name:"AdminInput")
// @gqlInput(name:"UserInput")
type Person struct {
	ID        string ` + "`gql:\"id,type:ID,ro\"`" + `
	Name      string
	SecretKey string ` + "`gql:\"secretKey,ro:Admin\"`" + `
}
`
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	parser := NewParser()
	if err := parser.Walk(tmpDir); err != nil {
		t.Fatalf("Parser walk failed: %v", err)
	}

	outFile := filepath.Join(tmpDir, "schema.graphqls")
	cfg := &Config{
		Packages:            []string{tmpDir},
		Output:              outFile,
		GenStrategy:         GenStrategySingle,
		UseGqlGenDirectives: true,
	}

	engine := NewGenerator(parser, cfg)
	if err := engine.Run(); err != nil {
		t.Fatalf("Generator run failed: %v", err)
	}

	content, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	schema := string(content)

	// Admin type should have id and secretKey (both read-only)
	if strings.Contains(schema, "type Admin") {
		adminStart := strings.Index(schema, "type Admin")
		adminEnd := strings.Index(schema[adminStart:], "}")
		adminType := schema[adminStart : adminStart+adminEnd]

		if !strings.Contains(adminType, "id") {
			t.Error("Admin type should contain id field (ro)")
		}
		if !strings.Contains(adminType, "secretKey") {
			t.Error("Admin type should contain secretKey field (ro:Admin)")
		}
	}

	// User type should have id but NOT secretKey
	if strings.Contains(schema, "type User") {
		userStart := strings.Index(schema, "type User")
		userEnd := strings.Index(schema[userStart:], "}")
		userType := schema[userStart : userStart+userEnd]

		if !strings.Contains(userType, "id") {
			t.Error("User type should contain id field (ro)")
		}
		if strings.Contains(userType, "secretKey") {
			t.Error("User type should NOT contain secretKey field (ro:Admin only)")
		}
	}

	// AdminInput should NOT have id or secretKey (both read-only)
	if strings.Contains(schema, "input AdminInput") {
		adminInputStart := strings.Index(schema, "input AdminInput")
		adminInputEnd := strings.Index(schema[adminInputStart:], "}")
		adminInputType := schema[adminInputStart : adminInputStart+adminInputEnd]

		if strings.Contains(adminInputType, "id") {
			t.Error("AdminInput should NOT contain id field (ro)")
		}
		if strings.Contains(adminInputType, "secretKey") {
			t.Error("AdminInput should NOT contain secretKey field (ro)")
		}
	}
}
