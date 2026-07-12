import { get } from '$lib/api';
import type { Holding, Portfolio } from '$lib/types';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ params }) => {
  const [portfolio, holdings] = await Promise.all([
    get<Portfolio>(`/portfolios/${params.id}`),
    get<Holding[]>(`/portfolios/${params.id}/holdings`),
  ]);
  return { portfolio, holdings };
};
