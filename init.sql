-- init.sql
BEGIN;

-- ========================
-- 1. Create ENUM Types
-- ========================
CREATE TYPE order_status AS ENUM (
    'pending', 
    'accepted', 
    'preparing', 
    'ready', 
    'delivered', 
    'cancelled'
);

CREATE TYPE unit_type AS ENUM (
    'g',       -- grams
    'ml',      -- milliliters
    'shots',   -- espresso shots
    'items'    -- discrete items
);

CREATE TYPE payment_method AS ENUM (
    'cash',
    'credit_card',
    'mobile_payment'
);

-- ========================
-- 2. Create Core Tables
-- ========================
CREATE TABLE menu_items (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    price DECIMAL(10,2) NOT NULL CHECK (price > 0),
    category TEXT[],
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE inventory (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    quantity DECIMAL(10,3) NOT NULL,
    unit unit_type NOT NULL,
    cost_per_unit DECIMAL(10,2),
    reorder_level DECIMAL(10,3),
    supplier_info JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW()
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE menu_item_ingredients (
    menu_item_id INTEGER REFERENCES menu_items(id) ON DELETE CASCADE,
    ingredient_id INTEGER REFERENCES inventory(id) ON DELETE RESTRICT,
    quantity DECIMAL(10,3) NOT NULL CHECK (quantity > 0),
    PRIMARY KEY (menu_item_id, ingredient_id)
);

CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    customer_id INTEGER REFERENCES customers(id) ON DELETE SET NULL,
    status order_status NOT NULL DEFAULT 'pending',
    payment_method payment_method,
    total_price DECIMAL(10,2) NOT NULL CHECK (total_price >= 0),
    special_instructions JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE order_items (
    id SERIAL PRIMARY KEY,
    order_id INTEGER REFERENCES orders(id) ON DELETE CASCADE,
    menu_item_id INTEGER REFERENCES menu_items(id),
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    customizations JSONB,
    price_at_order DECIMAL(10,2) NOT NULL CHECK (price_at_order >= 0)
);

CREATE TABLE customers (
    id SERIAL PRIMARY KEY,
    first_name TEXT NOT NULL,
    last_name TEXT NOT NULL,
    phone TEXT UNIQUE,
    email TEXT UNIQUE,
    loyalty_points INTEGER DEFAULT 0,
    preferences JSONB, -- e.g., {"favorite_drink": "latte", "milk_preference": "oat"}
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- ========================
-- 3. Create History Tables
-- ========================
CREATE TABLE order_status_history (
    id SERIAL PRIMARY KEY,
    order_id INTEGER REFERENCES orders(id) ON DELETE CASCADE,
    status order_status NOT NULL,
    changed_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE price_history (
    id SERIAL PRIMARY KEY,
    menu_item_id INTEGER REFERENCES menu_items(id) ON DELETE CASCADE,
    old_price DECIMAL(10,2) NOT NULL,
    new_price DECIMAL(10,2) NOT NULL,
    changed_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE inventory_transactions (
    id SERIAL PRIMARY KEY,
    ingredient_id INTEGER REFERENCES inventory(id) ON DELETE CASCADE,
    delta DECIMAL(10,3) NOT NULL,
    transaction_type TEXT NOT NULL CHECK (transaction_type IN ('purchase', 'usage', 'adjustment')),
    reference_id INTEGER, -- order_id or other reference
    notes TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- ========================
-- 4. Create Indexes
-- ========================
-- For performance on frequently queried columns
CREATE INDEX idx_orders_status ON orders(status);
CREATE INDEX idx_orders_created_at ON orders(created_at);
CREATE INDEX idx_menu_items_category ON menu_items USING GIN(category);

-- For full-text search
ALTER TABLE menu_items ADD COLUMN search_vector tsvector;
CREATE INDEX idx_menu_items_search ON menu_items USING GIN(search_vector);

-- For inventory management
CREATE INDEX idx_inventory_low_stock ON inventory(quantity) WHERE quantity < reorder_level;

-- ========================
-- 5. Create Triggers
-- ========================
-- Automatically update search vector
CREATE OR REPLACE FUNCTION menu_items_search_update() RETURNS TRIGGER AS $$
BEGIN
    NEW.search_vector = 
        setweight(to_tsvector('english', NEW.name), 'A') ||
        setweight(to_tsvector('english', NEW.description), 'B');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_menu_items_search_update
BEFORE INSERT OR UPDATE ON menu_items
FOR EACH ROW EXECUTE FUNCTION menu_items_search_update();

-- Track price changes
CREATE OR REPLACE FUNCTION log_price_change() RETURNS TRIGGER AS $$
BEGIN
    IF NEW.price <> OLD.price THEN
        INSERT INTO price_history (menu_item_id, old_price, new_price)
        VALUES (OLD.id, OLD.price, NEW.price);
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_log_price_change
AFTER UPDATE OF price ON menu_items
FOR EACH ROW EXECUTE FUNCTION log_price_change();

-- ========================
-- 6. Insert Sample Data
-- ========================
-- Inventory items
INSERT INTO inventory (name, quantity, unit, cost_per_unit, reorder_level) VALUES
('Espresso Beans', 5000, 'g', 0.02, 1000),
('Milk', 20000, 'ml', 0.01, 5000),
('Sugar', 10000, 'g', 0.005, 2000),
('Chocolate Syrup', 5000, 'ml', 0.03, 1000),
('Vanilla Extract', 1000, 'ml', 0.15, 200);

-- Menu items
INSERT INTO menu_items (name, description, price, category) VALUES
('Espresso', 'Strong black coffee', 2.50, ARRAY['coffee', 'hot']),
('Latte', 'Espresso with steamed milk', 3.50, ARRAY['coffee', 'hot', 'milk']),
('Iced Coffee', 'Cold brewed coffee', 3.00, ARRAY['coffee', 'cold']),
('Chocolate Cake', 'Rich chocolate dessert', 4.50, ARRAY['food', 'dessert']);

-- Menu item ingredients
INSERT INTO menu_item_ingredients VALUES
(1, 1, 7),  -- Espresso: 7g beans
(2, 1, 7),   -- Latte: 7g beans
(2, 2, 200), -- Latte: 200ml milk
(3, 1, 10),  -- Iced Coffee: 10g beans
(4, 3, 50),  -- Chocolate Cake: 50g sugar
(4, 4, 20);  -- Chocolate Cake: 20ml syrup

-- Orders with order items


-- ========================
-- 7. Update Search Vectors
-- ========================
UPDATE menu_items SET search_vector = 
    setweight(to_tsvector('english', name), 'A') ||
    setweight(to_tsvector('english', description), 'B');

COMMIT;