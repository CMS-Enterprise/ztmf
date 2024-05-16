import gql from 'graphql-tag'
import { client } from '@/main'

const FISMASYSTEMQUERY: any = gql`
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
// @ts-ignore
const pillarLoader = async () => {
  const { data } = await client.query(FISMASYSTEMQUERY)
  return data
}
export default pillarLoader
