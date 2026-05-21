import client from './client';

export const fetchAgentAssets = () => client.get('/hosts');

export const fetchAgentMetrics = (uuid, duration = '1h') =>
  client.get('/metrics', { params: { host_uuid: uuid, duration } });