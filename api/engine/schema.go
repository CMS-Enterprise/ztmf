package engine

var schema = `
schema {
  query: Query
}

type Query {
  fismasystems: [FismaSystem!]!
	fismasystem(fismaGUID: ID!): FismaSystem!
}

type FismaSystem {
	fismaGUID:             ID!
	fismaacronym:          String!
	fismaname:             String!
	fismasubsystem:        String!
	component:             String!
	groupacronym:          String!
	groupname:             String!
	divisionname:          String!
	datacenterenvironment: String!
	datacallcontact:       String!
	issoemail:             String!
}
`
