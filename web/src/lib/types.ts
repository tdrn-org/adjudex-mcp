export interface Portfolio {
	ID: string;
	Name: string;
	Description: string;
	Positions: Position[] | null;
	CreatedAt: string;
	UpdatedAt: string;
}

export interface Position {
	ID: string;
	Symbol: string;
	Quantity: number;
	EntryPrice: number;
	EntryDate: string;
	Notes: string;
}

export interface Holding {
	Position: Position;
	CurrentPrice: number;
	MarketValue: number;
	UnrealizedPnL: number;
	UnrealizedPnLPct: number;
}
