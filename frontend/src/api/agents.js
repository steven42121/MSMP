import axios from 'axios';

export function fetchAgentAssets() {
    return axios.get('/api/agents/assets/all');
}