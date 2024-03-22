package engine

var schema = `
schema {
  query: Query
}

type Query {
  fismasystems: [FismaSystem!]!
	fismasystem(fismasystemid: ID!): FismaSystem!
}

type FismaSystem {
	fismasystemid:             ID!
	fismaacronym:          String!
	fismaname:             String!
	fismasubsystem:        String
	component:             String
	groupacronym:          String
	groupname:             String
	divisionname:          String
	datacenterenvironment: String
	datacallcontact:       String
	issoemail:             String
}
`
