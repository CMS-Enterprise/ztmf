package graph

type RootResolver struct{}

const Schema = `
schema {
  query: Query
	mutation: Mutation
}

type Query {
  fismasystems(fismaacronym: String): [FismaSystem!]!
	fismasystem(fismasystemid: ID!):    FismaSystem!
	functions:                          [Function!]!
	function(functionid: ID!):          Function!
	users:                              [User!]!
	user(userid: ID!):                  User!
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

type User {
	userid:         ID!
	email:          String!
	fullname:       String!
	role:           String!
	fismasystemids: [Int]!
}

type Mutation {
  saveUser(userid: ID, email: String!, fullname: String!, role: String!): UserMutationResponse!
  saveFunctionScore(scoreid: ID, fismasystemid: Int!, functionid: Int!, score: Float!, notes: String): FunctionScoreMutationReponse!
	assignFismaSystems(userid: String!, fismasystemids: [Int!]!): UserMutationResponse!
	unassignFismaSystems(userid: String!, fismasystemids: [Int!]!): UserMutationResponse!
}

interface Response {
	code:    Int!
	message: String!
}

type UserMutationResponse implements Response {
	code:    Int!
	message: String!
	user:    User
}

type FunctionScoreMutationReponse implements Response {
  code:          Int!
  message:       String!
  functionscore: FunctionScore
}
`
