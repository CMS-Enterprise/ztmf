import gql from 'graphql-tag'

const FISMASYSTEMQUERY = gql`
  query getFismaSystem($id: ID!) {
    fismasystem(fismasystemid: $id) {
      fismasystemid
      fismauid
      fismaacronym
      functionscores {
        # datecalculated
        score
        function {
          pillar
          name
        }
      }
    }
  }
`

export interface functionInfo {
  pillar: string
  name: string
}
export interface functionscores {
  score: number
  function: functionInfo
}

export interface fismaSystem {
  fismasystemid: string
  fismauid: string
  fismaacronym: string
  functionscores: functionscores[]
}
export default FISMASYSTEMQUERY
