import express, { Express, Request, Response } from 'express';

if (process.env.NODE_ENV !== 'production') {
  require('dotenv').config();
}

const app: Express = express();
const port = process.env.PORT;

app.disable('x-powered-by');

app.get('*', (req: Request, res: Response) => {
  res.send('ZTMF Scoring');
});

app.listen(port, () => {
  console.log(`Server is running on ${port}`);
});
