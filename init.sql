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

CREATE TYPE transaction_type AS ENUM (
    'order_usage',
    'order_deletion',
    'adjustment'
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
    transaction_type transaction_type NOT NULL,
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

-- Insert 20 inventory items
INSERT INTO inventory (name, quantity, unit, cost_per_unit, reorder_level, supplier_info) VALUES
('Espresso Beans', 5000, 'g', 0.02, 1000, '{"supplier": "Bean Co", "contact": "555-1001"}'),
('Arabica Beans', 3000, 'g', 0.03, 1500, '{"supplier": "Premium Beans", "contact": "555-1002"}'),
('Robusta Beans', 4000, 'g', 0.025, 1200, '{"supplier": "Bean Co", "contact": "555-1001"}'),
('Milk', 20000, 'ml', 0.01, 5000, '{"supplier": "Dairy Farms", "contact": "555-2001"}'),
('Oat Milk', 15000, 'ml', 0.015, 4000, '{"supplier": "Plant Co", "contact": "555-2002"}'),
('Almond Milk', 10000, 'ml', 0.018, 3000, '{"supplier": "Plant Co", "contact": "555-2002"}'),
('Sugar', 10000, 'g', 0.005, 2000, '{"supplier": "Sweet Inc", "contact": "555-3001"}'),
('Brown Sugar', 8000, 'g', 0.006, 1500, '{"supplier": "Sweet Inc", "contact": "555-3001"}'),
('Chocolate Syrup', 5000, 'ml', 0.03, 1000, '{"supplier": "Choc Co", "contact": "555-4001"}'),
('Caramel Syrup', 4500, 'ml', 0.032, 1000, '{"supplier": "Sweet Inc", "contact": "555-3001"}'),
('Vanilla Extract', 1000, 'ml', 0.15, 200, '{"supplier": "Flavor Co", "contact": "555-5001"}'),
('Cinnamon Powder', 2000, 'g', 0.08, 500, '{"supplier": "Spice World", "contact": "555-6001"}'),
('Whipped Cream', 3000, 'ml', 0.04, 800, '{"supplier": "Dairy Farms", "contact": "555-2001"}'),
('Ice Cubes', 50000, 'items', 0.001, 10000, '{"supplier": "Ice Co", "contact": "555-7001"}'),
('Paper Cups (12oz)', 1000, 'items', 0.05, 300, '{"supplier": "Packaging Co", "contact": "555-8001"}'),
('Lids', 1200, 'items', 0.03, 400, '{"supplier": "Packaging Co", "contact": "555-8001"}'),
('Straws', 2000, 'items', 0.01, 500, '{"supplier": "Packaging Co", "contact": "555-8001"}'),
('Coffee Filters', 500, 'items', 0.02, 100, '{"supplier": "Brew Co", "contact": "555-9001"}'),
('Chocolate Chips', 2500, 'g', 0.07, 500, '{"supplier": "Choc Co", "contact": "555-4001"}'),
('Biscotti', 200, 'items', 0.25, 50, '{"supplier": "Bakery Co", "contact": "555-10001"}');

-- Insert 10 menu items with different categories
INSERT INTO menu_items (name, description, price, category, is_active) VALUES
('Espresso', 'Strong black coffee made from premium beans', 2.50, ARRAY['coffee', 'hot'], true),
('Double Espresso', 'Twice the coffee, twice the energy', 3.50, ARRAY['coffee', 'hot'], true),
('Americano', 'Espresso with hot water', 3.00, ARRAY['coffee', 'hot'], true),
('Latte', 'Espresso with steamed milk', 3.75, ARRAY['coffee', 'hot', 'milk'], true),
('Cappuccino', 'Espresso with equal parts steamed milk and foam', 4.00, ARRAY['coffee', 'hot', 'milk'], true),
('Iced Coffee', 'Cold brewed coffee served over ice', 3.50, ARRAY['coffee', 'cold'], true),
('Iced Latte', 'Espresso with cold milk over ice', 4.25, ARRAY['coffee', 'cold', 'milk'], true),
('Hot Chocolate', 'Rich chocolate drink with steamed milk', 3.75, ARRAY['hot', 'chocolate'], true),
('Chocolate Cake', 'Rich chocolate dessert with layers of ganache', 5.50, ARRAY['food', 'dessert'], true),
('Blueberry Muffin', 'Fresh muffin with blueberries', 3.25, ARRAY['food', 'bakery'], false);  -- One inactive item for testing

-- Insert menu item ingredients
INSERT INTO menu_item_ingredients VALUES
(1, 1, 7),   -- Espresso: 7g beans
(2, 1, 14),  -- Double Espresso: 14g beans
(3, 1, 7),   -- Americano: 7g beans
(3, 4, 150), -- Americano: 150ml hot water
(4, 1, 7),   -- Latte: 7g beans
(4, 4, 200), -- Latte: 200ml milk
(5, 1, 7),   -- Cappuccino: 7g beans
(5, 4, 100), -- Cappuccino: 100ml milk
(6, 1, 10),  -- Iced Coffee: 10g beans
(6, 14, 10), -- Iced Coffee: 10 ice cubes
(7, 1, 7),   -- Iced Latte: 7g beans
(7, 4, 200), -- Iced Latte: 200ml milk
(7, 14, 10), -- Iced Latte: 10 ice cubes
(8, 9, 30),  -- Hot Chocolate: 30ml chocolate syrup
(8, 4, 250), -- Hot Chocolate: 250ml milk
(9, 19, 50), -- Chocolate Cake: 50g chocolate chips
(9, 7, 30),  -- Chocolate Cake: 30g sugar
(9, 4, 50),  -- Chocolate Cake: 50ml milk
(10, 7, 20), -- Blueberry Muffin: 20g sugar
(10, 4, 30); -- Blueberry Muffin: 30ml milk

-- Insert customers
INSERT INTO customers (first_name, last_name, phone, email, loyalty_points, preferences) VALUES
('John', 'Smith', '555-0101', 'john.smith@email.com', 120, '{"favorite_drink": "latte", "milk_preference": "oat"}'),
('Emily', 'Johnson', '555-0102', 'emily.j@email.com', 75, '{"favorite_drink": "cappuccino", "milk_preference": "almond"}'),
('Michael', 'Williams', '555-0103', 'michael.w@email.com', 200, '{"favorite_drink": "espresso"}'),
('Sarah', 'Brown', '555-0104', 'sarah.b@email.com', 30, '{"favorite_drink": "iced latte"}'),
('David', 'Jones', '555-0105', 'david.j@email.com', 0, '{"favorite_drink": "americano"}'),
('Jessica', 'Garcia', '555-0106', 'jessica.g@email.com', 150, '{"favorite_drink": "hot chocolate", "whipped_cream": true}'),
('Robert', 'Miller', '555-0107', 'robert.m@email.com', 50, '{"favorite_drink": "double espresso"}'),
('Jennifer', 'Davis', '555-0108', 'jennifer.d@email.com', 80, '{"favorite_drink": "latte", "extra_shot": true}'),
('Thomas', 'Rodriguez', '555-0109', 'thomas.r@email.com', 10, '{"favorite_drink": "iced coffee"}'),
('Lisa', 'Martinez', '555-0110', 'lisa.m@email.com', 95, '{"favorite_drink": "cappuccino", "cinnamon": true}');

-- Insert 30 orders with different statuses (spanning 6 months)
INSERT INTO orders (customer_id, status, payment_method, total_price, special_instructions, created_at) VALUES
(1, 'delivered', 'credit_card', 7.25, '{"notes": "Extra hot"}', NOW() - INTERVAL '5 months'),
(2, 'delivered', 'mobile_payment', 10.50, '{"notes": "No foam"}', NOW() - INTERVAL '5 months' + INTERVAL '2 days'),
(3, 'delivered', 'cash', 5.50, NULL, NOW() - INTERVAL '4 months'),
(4, 'delivered', 'credit_card', 8.75, '{"notes": "Light ice"}', NOW() - INTERVAL '4 months' + INTERVAL '1 week'),
(5, 'delivered', 'mobile_payment', 6.00, NULL, NOW() - INTERVAL '3 months'),
(6, 'delivered', 'cash', 12.25, '{"notes": "Extra whipped cream"}', NOW() - INTERVAL '3 months' + INTERVAL '3 days'),
(7, 'delivered', 'credit_card', 9.50, NULL, NOW() - INTERVAL '2 months'),
(8, 'delivered', 'mobile_payment', 7.75, '{"notes": "Sugar on side"}', NOW() - INTERVAL '2 months' + INTERVAL '5 days'),
(9, 'delivered', 'cash', 4.25, NULL, NOW() - INTERVAL '1 month'),
(10, 'delivered', 'credit_card', 11.00, '{"notes": "Caramel drizzle"}', NOW() - INTERVAL '1 month' + INTERVAL '2 days'),
(1, 'delivered', 'mobile_payment', 8.50, NULL, NOW() - INTERVAL '3 weeks'),
(2, 'delivered', 'cash', 6.75, '{"notes": "Extra shot"}', NOW() - INTERVAL '2 weeks'),
(3, 'delivered', 'credit_card', 10.25, NULL, NOW() - INTERVAL '10 days'),
(4, 'ready', 'mobile_payment', 7.00, '{"notes": "For pickup"}', NOW() - INTERVAL '5 days'),
(5, 'ready', 'cash', 5.25, NULL, NOW() - INTERVAL '3 days'),
(6, 'preparing', 'credit_card', 9.75, '{"notes": "Allergy: nuts"}', NOW() - INTERVAL '2 days'),
(7, 'preparing', 'mobile_payment', 8.00, NULL, NOW() - INTERVAL '1 day'),
(8, 'accepted', 'cash', 6.50, '{"notes": "Contactless delivery"}', NOW() - INTERVAL '12 hours'),
(9, 'accepted', 'credit_card', 11.50, NULL, NOW() - INTERVAL '6 hours'),
(10, 'pending', 'mobile_payment', 7.25, '{"notes": "Call on arrival"}', NOW() - INTERVAL '1 hour'),
(1, 'pending', 'cash', 5.75, NULL, NOW() - INTERVAL '30 minutes'),
(2, 'pending', 'credit_card', 10.00, '{"notes": "Gift for Sarah"}', NOW() - INTERVAL '15 minutes'),
(3, 'cancelled', 'mobile_payment', 8.25, NULL, NOW() - INTERVAL '1 month' + INTERVAL '1 week'),
(4, 'cancelled', 'cash', 6.50, '{"notes": "Wrong address"}', NOW() - INTERVAL '2 weeks' + INTERVAL '3 days'),
(5, 'cancelled', 'credit_card', 9.00, NULL, NOW() - INTERVAL '5 days'),
(6, 'delivered', 'mobile_payment', 12.50, '{"notes": "Birthday surprise"}', NOW() - INTERVAL '1 day'),
(7, 'delivered', 'cash', 7.75, NULL, NOW() - INTERVAL '8 hours'),
(8, 'delivered', 'credit_card', 11.25, '{"notes": "Corporate order"}', NOW() - INTERVAL '4 hours'),
(9, 'ready', 'mobile_payment', 6.50, NULL, NOW() - INTERVAL '2 hours'),
(10, 'preparing', 'cash', 9.25, '{"notes": "Fragile contents"}', NOW() - INTERVAL '45 minutes');

-- Insert order items
INSERT INTO order_items (order_id, menu_item_id, quantity, customizations, price_at_order) VALUES
-- Order 1
(1, 4, 1, '{"milk": "oat"}', 3.75),
(1, 9, 1, NULL, 5.50),
-- Order 2
(2, 2, 1, NULL, 3.50),
(2, 4, 2, '{"extra_shot": true}', 3.75),
-- Order 3
(3, 1, 2, NULL, 2.50),
-- Order 4
(4, 7, 1, '{"ice": "light"}', 4.25),
(4, 10, 1, NULL, 3.25),
-- Order 5
(5, 3, 2, NULL, 3.00),
-- Order 6
(6, 8, 1, '{"whipped_cream": "extra"}', 3.75),
(6, 9, 1, NULL, 5.50),
(6, 10, 1, NULL, 3.25),
-- Order 7
(7, 5, 2, NULL, 4.00),
-- Order 8
(8, 4, 1, '{"sugar": "on side"}', 3.75),
(8, 10, 1, NULL, 3.25),
-- Order 9
(9, 6, 1, NULL, 3.50),
-- Order 10
(10, 4, 1, '{"caramel": "drizzle"}', 3.75),
(10, 5, 1, NULL, 4.00),
(10, 10, 1, NULL, 3.25),
-- [Continuing with similar patterns for orders 11-30]
-- Order 11
(11, 4, 2, NULL, 3.75),
-- Order 12
(12, 2, 1, '{"extra_shot": true}', 3.50),
(12, 6, 1, NULL, 3.50),
-- Order 13
(13, 5, 2, NULL, 4.00),
(13, 9, 1, NULL, 5.50),
-- Order 14
(14, 3, 1, NULL, 3.00),
(14, 7, 1, NULL, 4.25),
-- Order 15
(15, 1, 1, NULL, 2.50),
(15, 10, 1, NULL, 3.25),
-- Order 16
(16, 5, 1, '{"milk": "almond"}', 4.00),
(16, 8, 1, NULL, 3.75),
(16, 9, 1, NULL, 5.50),
-- Order 17
(17, 4, 2, NULL, 3.75),
-- Order 18
(18, 3, 1, NULL, 3.00),
(18, 6, 1, NULL, 3.50),
(18, 10, 1, NULL, 3.25),
-- Order 19
(19, 5, 2, NULL, 4.00),
(19, 9, 1, NULL, 5.50),
-- Order 20
(20, 4, 1, '{"notes": "Call on arrival"}', 3.75),
(20, 7, 1, NULL, 4.25),
-- Order 21
(21, 1, 1, NULL, 2.50),
(21, 10, 1, NULL, 3.25),
-- Order 22
(22, 5, 1, '{"notes": "Gift for Sarah"}', 4.00),
(22, 8, 1, NULL, 3.75),
-- Order 23
(23, 4, 2, NULL, 3.75),
(23, 6, 1, NULL, 3.50),
-- Order 24
(24, 3, 1, '{"notes": "Wrong address"}', 3.00),
(24, 7, 1, NULL, 4.25),
-- Order 25
(25, 2, 1, NULL, 3.50),
(25, 9, 1, NULL, 5.50),
-- Order 26
(26, 5, 2, '{"notes": "Birthday surprise"}', 4.00),
(26, 8, 1, NULL, 3.75),
-- Order 27
(27, 4, 1, NULL, 3.75),
(27, 10, 1, NULL, 3.25),
-- Order 28
(28, 5, 1, '{"notes": "Corporate order"}', 4.00),
(28, 7, 1, NULL, 4.25),
(28, 9, 1, NULL, 5.50),
-- Order 29
(29, 3, 1, NULL, 3.00),
(29, 6, 1, NULL, 3.50),
-- Order 30
(30, 4, 1, '{"notes": "Fragile contents"}', 3.75),
(30, 8, 1, NULL, 3.75);

-- Insert order status history (showing transitions)
INSERT INTO order_status_history (order_id, status, changed_at) VALUES
-- Order 1 (delivered)
(1, 'pending', NOW() - INTERVAL '5 months'),
(1, 'accepted', NOW() - INTERVAL '5 months' + INTERVAL '5 minutes'),
(1, 'preparing', NOW() - INTERVAL '5 months' + INTERVAL '10 minutes'),
(1, 'ready', NOW() - INTERVAL '5 months' + INTERVAL '20 minutes'),
(1, 'delivered', NOW() - INTERVAL '5 months' + INTERVAL '35 minutes'),
-- Order 6 (delivered with special request)
(6, 'pending', NOW() - INTERVAL '3 months' + INTERVAL '3 days'),
(6, 'accepted', NOW() - INTERVAL '3 months' + INTERVAL '3 days' + INTERVAL '5 minutes'),
(6, 'preparing', NOW() - INTERVAL '3 months' + INTERVAL '3 days' + INTERVAL '15 minutes'),
(6, 'ready', NOW() - INTERVAL '3 months' + INTERVAL '3 days' + INTERVAL '30 minutes'),
(6, 'delivered', NOW() - INTERVAL '3 months' + INTERVAL '3 days' + INTERVAL '45 minutes'),
-- Order 16 (currently preparing)
(16, 'pending', NOW() - INTERVAL '2 days'),
(16, 'accepted', NOW() - INTERVAL '2 days' + INTERVAL '10 minutes'),
(16, 'preparing', NOW() - INTERVAL '2 days' + INTERVAL '25 minutes'),
-- Order 20 (currently pending)
(20, 'pending', NOW() - INTERVAL '1 hour'),
-- Order 23 (cancelled)
(23, 'pending', NOW() - INTERVAL '1 month' + INTERVAL '1 week'),
(23, 'accepted', NOW() - INTERVAL '1 month' + INTERVAL '1 week' + INTERVAL '5 minutes'),
(23, 'cancelled', NOW() - INTERVAL '1 month' + INTERVAL '1 week' + INTERVAL '15 minutes'),
-- Order 30 (currently preparing)
(30, 'pending', NOW() - INTERVAL '45 minutes'),
(30, 'accepted', NOW() - INTERVAL '45 minutes' + INTERVAL '5 minutes'),
(30, 'preparing', NOW() - INTERVAL '45 minutes' + INTERVAL '10 minutes');

-- Insert price history (spanning several months)
INSERT INTO price_history (menu_item_id, old_price, new_price, changed_at) VALUES
-- Espresso price changes
(1, 2.00, 2.25, NOW() - INTERVAL '6 months'),
(1, 2.25, 2.50, NOW() - INTERVAL '3 months'),
-- Latte price changes
(4, 3.25, 3.50, NOW() - INTERVAL '5 months'),
(4, 3.50, 3.75, NOW() - INTERVAL '2 months'),
-- Iced Coffee price changes
(6, 3.00, 3.25, NOW() - INTERVAL '4 months'),
(6, 3.25, 3.50, NOW() - INTERVAL '1 month'),
-- Chocolate Cake price changes
(9, 5.00, 5.25, NOW() - INTERVAL '3 months'),
(9, 5.25, 5.50, NOW() - INTERVAL '1 month'),
-- Cappuccino price changes
(5, 3.75, 4.00, NOW() - INTERVAL '2 months');

-- Insert inventory transactions (stock movements)
INSERT INTO inventory_transactions (ingredient_id, delta, transaction_type, reference_id, notes) VALUES
-- Order usage
(1, -14, 'order_usage', 1, 'Order #1 - 2 espressos'),
(4, -200, 'order_usage', 1, 'Order #1 - 1 latte'),
(9, -50, 'order_usage', 1, 'Order #1 - 1 chocolate cake'),
-- Order usage
(1, -7, 'order_usage', 2, 'Order #2 - 1 double espresso'),
(4, -400, 'order_usage', 2, 'Order #2 - 2 lattes'),
-- Adjustments
(4, 5000, 'adjustment', NULL, 'Milk delivery'),
(1, 2000, 'adjustment', NULL, 'Beans delivery'),
-- Order deletion (cancelled order)
(1, 14, 'order_deletion', 23, 'Order #23 cancelled'),
(4, 400, 'order_deletion', 23, 'Order #23 cancelled'),
-- More transactions
(9, -30, 'order_usage', 10, 'Order #10 - 1 hot chocolate'),
(4, -250, 'order_usage', 10, 'Order #10 - 1 hot chocolate'),
(19, -50, 'order_usage', 9, 'Order #9 - 1 chocolate cake'),
(7, -30, 'order_usage', 9, 'Order #9 - 1 chocolate cake'),
(4, -50, 'order_usage', 9, 'Order #9 - 1 chocolate cake');

-- ========================
-- 7. Update Search Vectors
-- ========================
UPDATE menu_items SET search_vector = 
    setweight(to_tsvector('english', name), 'A') ||
    setweight(to_tsvector('english', description), 'B');

COMMIT;