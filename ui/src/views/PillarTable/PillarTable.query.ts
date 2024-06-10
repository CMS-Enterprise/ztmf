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
        notes
        function {
          pillar
          name
          description
          traditional
          initial
          advanced
          optimal
        }
      }
    }
  }
`

export function determineLevel(
  score: number,
  fn: functionInfo
): [string, string] {
  let fnLevel: string = ''
  let description: string = ''
  if (score >= 1 && score <= 1.74) {
    fnLevel = 'Traditional'
    description = fn.traditional!
  } else if (score >= 1.75 && score <= 2.74) {
    fnLevel = 'Initial'
    description = fn.initial!
  } else if (score >= 2.75 && score <= 3.65) {
    fnLevel = 'Advanced'
    description = fn.advanced!
  } else if (score >= 3.66) {
    fnLevel = 'Optimal'
    description = fn.optimal!
  }
  return [fnLevel, description]
}

export interface functionInfo {
  pillar: string
  name: string
  description: string
  traditional?: string
  initial?: string
  advanced?: string
  optimal?: string
}
export interface functionscores {
  score: number
  notes: string
  function: functionInfo
}

export interface fismaSystem {
  fismasystemid: string
  fismauid: string
  fismaacronym: string
  functionscores: functionscores[]
}
export default FISMASYSTEMQUERY
