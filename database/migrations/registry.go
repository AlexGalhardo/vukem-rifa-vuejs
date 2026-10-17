// Package migrations holds the schema migrations.
package migrations

import "github.com/AlexGalhardo/vukem/schema"

// All lists the migrations in order; vukem make:migration appends here.
func All() []schema.Migration {
	return []schema.Migration{
		m20261017124833CreateUsersTable(),
		m20261017124834CreateAuthTables(),
		m20261017124835CreateSocialAccountsTable(),
	}
}
