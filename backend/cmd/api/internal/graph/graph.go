package graph

const Schema = `
schema {
  query: Query
}

type Query {
  fismasystems(fismaacronym: String): [FismaSystem!]!
	fismasystem(fismasystemid: ID!): FismaSystem!
	functions: [Function!]!
	function(functionid: ID!): Function!
}

type FismaSystem {
	fismasystemid:         ID!
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
	functionscores:        [FunctionScore!]!
}

type Function {
	functionid:            ID!
  pillar:                String
  name:                  String
  description:           String
  traditional:           String
  initial:               String
  advanced:              String
  optimal:               String
  datacenterenvironment: String
}

type FunctionScore {
	scoreid:        ID!
	fismasystemid:  Int!
	functionid:     Int!
	datecalculated: Float!
	score:          Float!
	notes:          String
	function:       Function!
}
`

type RootResolver struct{}
