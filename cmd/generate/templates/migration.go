package templates

var MigUp = `CREATE TABLE IF NOT EXISTS {{.PluralSnake}} (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
{{- range .Fields}}
    {{.SnakeName}} {{.SQLType}},
{{- end}}
    is_active BOOLEAN NOT NULL DEFAULT true,
    is_deletable BOOLEAN NOT NULL DEFAULT true,
    record_version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now(),
    deleted_at TIMESTAMP
);

CREATE TRIGGER update_{{.PluralSnake}}_updated_at
    BEFORE UPDATE ON {{.PluralSnake}}
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER increment_{{.PluralSnake}}_record_version
    BEFORE UPDATE ON {{.PluralSnake}}
    FOR EACH ROW EXECUTE FUNCTION increment_record_version();
`

var MigDown = `DROP TRIGGER IF EXISTS increment_{{.PluralSnake}}_record_version ON {{.PluralSnake}};
DROP TRIGGER IF EXISTS update_{{.PluralSnake}}_updated_at ON {{.PluralSnake}};
DROP TABLE IF EXISTS {{.PluralSnake}};
`
