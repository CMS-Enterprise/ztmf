package graph

type RootResolver struct{}

type Response struct {
	Code    int32
	Message string
}

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
  createUser(email: String!, fullname: String!, role: String!): CreateUserResponse!
  saveFunctionScore(scoreid: ID, fismasystemid: Int!, functionid: Int!, score: Float!, notes: String): SaveFunctionScoreReponse!
	assignFismaSystems(userid: String!, fismasystemids: [Int!]!): AssignFismaSystemsReponse!
}

interface Response {
	code:    Int!
	message: String!
}

type CreateUserResponse implements Response {
	code:    Int!
	message: String!
	user:    User
}

type SaveFunctionScoreReponse implements Response {
  code:          Int!
  message:       String!
  functionscore: FunctionScore
}

type AssignFismaSystemsReponse implements Response {
	code:    Int!
	message: String!
	user:    User
}
`
