import type { Portfolio, Holding } from './types';

const BASE = '/api/v1';

async function fetchJSON<T>(url: string, init?: RequestInit): Promise<T> {
	const r = await fetch(url, init);
	if (!r.ok) {
		const err = await r.json().catch(() => ({ error: r.statusText }));
		throw new Error(err.error || `HTTP ${r.status}`);
	}
	return r.json() as Promise<T>;
}

export function listPortfolios(): Promise<Portfolio[]> {
	return fetchJSON<Portfolio[]>(`${BASE}/portfolios`);
}

export function getPortfolio(id: string): Promise<Portfolio> {
	return fetchJSON<Portfolio>(`${BASE}/portfolios/${encodeURIComponent(id)}`);
}

export function createPortfolio(name: string, description: string): Promise<Portfolio> {
	return fetchJSON<Portfolio>(`${BASE}/portfolios`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ name, description })
	});
}

export function deletePortfolio(id: string): Promise<void> {
	return fetchJSON<void>(`${BASE}/portfolios/${encodeURIComponent(id)}`, { method: 'DELETE' });
}

export function getHoldings(id: string): Promise<Holding[]> {
	return fetchJSON<Holding[]>(`${BASE}/portfolios/${encodeURIComponent(id)}/holdings`);
}
