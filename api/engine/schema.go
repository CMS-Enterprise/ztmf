package engine

var schema = `
schema {
  query: Query
}

type Query {
  fismasystems: [FismaSystem!]!
	fismasystem(fismasystemid: ID!): FismaSystem!
	functions: [Function!]!
	function(functionid: ID!): Function!
}

type FismaSystem {
	fismasystemid:             ID!
	fismauid:          		 String!
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

type Function {
	functionid:     ID!
  pillar:      String
  name:        String
  description: String
  traditional: String
  initial:     String
  advanced:    String
  optimal:     String
  environment: String
}
`
