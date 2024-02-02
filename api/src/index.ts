import express, { Express, Request, Response } from 'express';
import { SecretsManagerClient, GetSecretValueCommand } from "@aws-sdk/client-secrets-manager"
import { Server, ServerOptions } from 'https';
import * as dotenv from 'dotenv';
dotenv.config();

async function run() {
  const app: Express = express();
  const PORT = process.env.PORT;

  app.disable('x-powered-by');

  app.get('*', (req: Request, res: Response) => {
    res.send('ZTMF Scoring');
  });

  // pull TLS cert and private key from secrets manager
  const client = new SecretsManagerClient();

  const certInput = { SecretId: process.env.CERT_SEC };
  const certCommand = new GetSecretValueCommand(certInput);
  const certResponse = await client.send(certCommand);

  const keyInput = { SecretId: process.env.KEY_SEC };
  const keyCommand = new GetSecretValueCommand(keyInput);
  const keyResponse = await client.send(keyCommand);

  // configured node to run with TLS
  const serverOptions = <ServerOptions>{
    cert: certResponse.SecretString,
    key: keyResponse.SecretString,
    maxVersion: 'TLSv1.3',
    minVersion: 'TLSv1.3'
  }

  const server = new Server(serverOptions, app);

  server.listen(PORT, () => {
    console.log(`HTTPS server listening on ${PORT}`);
  });
}

run();
