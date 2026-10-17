// Package migrations holds the schema migrations.
package migrations

import "github.com/AlexGalhardo/vukem/schema"

// All lists the migrations in order; vukem make:migration appends here.
func All() []schema.Migration {
	return []schema.Migration{
		m20261017124833CreateUsersTable(),
		m20261017124834CreateAuthTables(),
		m20261017124835CreateSocialAccountsTable(),
		m20261017130000AddIsAdminToUsers(),
		m20261017140000CreateRafflesTables(),
		m20261017150000CreateReservationsTable(),
		m20261017160000CreatePaymentsTables(),
		m20261017170000CreateDrawsTable(),
		m20261017190000AddLocaleToReservations(),
		m20261017210000AddTokenSealedToReservations(),
		m20261017230000AddRevokedAtToAccessTokens(),
		m20261017240000AddOriginAndBuyerUserToReservations(),
	}
}
