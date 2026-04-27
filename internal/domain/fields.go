package domain

const (
	AppFieldPrefix = "z4j_"

	FieldNodeUID                  = AppFieldPrefix + "node_uid"
	FieldRelUID                   = AppFieldPrefix + "rel_uid"
	FieldOrigin                   = AppFieldPrefix + "origin"
	FieldCreatedAt                = AppFieldPrefix + "created_at"
	FieldUpdatedAt                = AppFieldPrefix + "updated_at"
	FieldExpiresAt                = AppFieldPrefix + "expires_at"
	FieldNodeTemplateHashes       = AppFieldPrefix + "template_hashes"
	FieldRelationshipTemplateHash = AppFieldPrefix + "template_hash"
)
