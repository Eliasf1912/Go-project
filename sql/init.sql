-- ============================================================
-- Boutique CLI — schéma de base de données (init.sql, Phase 0)
-- PostgreSQL (via conteneur Docker), compatible database/sql
-- Ordre des CREATE TABLE = ordre des dépendances (FK)
-- À exécuter avant seed.sql
--
-- v5 — les VALEURS des statuts repassent en anglais (pending, paid,
--   shipping, delivered, cancelled / open, paid), par cohérence avec
--   les noms de tables/colonnes déjà en anglais. Les libellés affichés
--   à l'utilisateur (« en attente », « payée »...) restent en français,
--   traduits côté Go au moment de l'affichage — ce n'est plus la même
--   valeur que celle stockée/échangée en JSON.
--
-- v4 — aligné sur DECISIONS.md (contrat de l'API, partie 2) :
--   - les VALEURS des statuts de commande/panier sont en français
--     (en_attente, payee, en_cours_livraison, livree, annulee /
--     ouvert, paye), telles qu'exposées telles quelles en JSON —
--     seuls les noms de tables/colonnes restent en anglais
--
-- v3 — aligné sur ARCHITECTURE.md (section 7 "Modèle de données") :
--   - noms de tables/colonnes en anglais (cohérent avec le code Go,
--     les DTO JSON et les routes de l'API)
--   - dates en TIMESTAMPTZ (UTC en base, converties en heure de Paris
--     uniquement à l'affichage)
--   - référence métier sur 6 caractères [A-Z0-9] : PDT-XXXXXX,
--     BSK-XXXXXX, CMD-XXXXXX (10 caractères au total)
--   - sessions : le jeton lui-même est la clé primaire ; la ligne est
--     supprimée à la déconnexion (pas de colonne "révoquée")
--   - cart_items : clé primaire composée (cart_id, product_id), pas
--     de colonne id — applique directement "un produit par panier"
--   - price_ht > 0 (CHECK strict) ; price_ttc = colonne générée par
--     la base (ROUND(price_ht * 1.2), TVA fixe 20 %), jamais recalculée
--     en Go
--   - order_items recopie aussi le NOM du produit, en plus du prix,
--     au moment de la commande (pas seulement le prix figé)
--   - produit déjà commandé -> archivé (active = false), jamais
--     supprimé ; suppression réelle uniquement si jamais commandé
--   - utilisateur avec commandes -> anonymisé (deleted_at), jamais
--     supprimé ; suppression réelle uniquement si aucune commande
--   - aucune table "payments" : le paiement ne persiste aucune donnée
--     bancaire, même réduite (ni numéro, ni 4 derniers chiffres, ni
--     marque) — seul order.status passe à 'payee'. À confirmer avec
--     Elias : si vous voulez quand même tracer un montant/une date de
--     paiement, je peux ajouter deux colonnes sur `orders`
--     (paid_amount, paid_at) sans réintroduire de table dédiée.
--
-- Note SQLite : SQLite n'a pas de type ENUM natif, pas de SERIAL, pas
-- de TIMESTAMPTZ, et pas de colonnes générées STORED avant la version
-- 3.31. Dites-le-moi si vous partez sur SQLite, je prépare une variante.
-- ============================================================

-- ------------------------------------------------------------
-- Types énumérés
-- ------------------------------------------------------------

CREATE TYPE user_role AS ENUM ('client', 'admin');
CREATE TYPE cart_status AS ENUM ('open', 'paid');
CREATE TYPE order_status AS ENUM (
    'pending',
    'paid',
    'shipping',
    'delivered',
    'cancelled'
);
-- Ces valeurs sont exposées telles quelles dans le JSON de l'API.
-- Les libellés en français ("En attente", "Payée"...) sont un
-- affichage géré côté Go (une seule fonction de traduction), pas la
-- valeur stockée ni échangée en JSON.

-- ------------------------------------------------------------
-- users
-- ------------------------------------------------------------

CREATE TABLE users (
    id                          SERIAL PRIMARY KEY,
    email                       VARCHAR(255) NOT NULL UNIQUE,
    password_hash               VARCHAR(255) NOT NULL, -- bcrypt, jamais en clair
    role                        user_role NOT NULL DEFAULT 'client',
    confirmed                   BOOLEAN NOT NULL DEFAULT false,
    confirmation_code           VARCHAR(64),            -- code unique, NULL une fois utilisé
    password_reset_code         VARCHAR(64),
    password_reset_expires_at   TIMESTAMPTZ,
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                  TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at                  TIMESTAMPTZ             -- date d'anonymisation (compte conservé,
                                                          -- données personnelles écrasées par l'app)
);

-- ------------------------------------------------------------
-- sessions (jeton de connexion = clé primaire)
-- ------------------------------------------------------------

CREATE TABLE sessions (
    token       VARCHAR(255) PRIMARY KEY,               -- généré avec crypto/rand
    user_id     INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at  TIMESTAMPTZ NOT NULL
);

-- ------------------------------------------------------------
-- categories
-- ------------------------------------------------------------

CREATE TABLE categories (
    id          SERIAL PRIMARY KEY,
    name        VARCHAR(100) NOT NULL UNIQUE CHECK (name = lower(name)),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ------------------------------------------------------------
-- products
-- ------------------------------------------------------------

CREATE TABLE products (
    id            SERIAL PRIMARY KEY,
    reference     VARCHAR(10) NOT NULL UNIQUE,          -- format PDT-XXXXXX
    name          VARCHAR(255) NOT NULL,
    description   TEXT,
    price_ht      INTEGER NOT NULL CHECK (price_ht > 0), -- centimes
    price_ttc     INTEGER GENERATED ALWAYS AS (ROUND(price_ht * 1.2)::INTEGER) STORED,
                                                           -- TVA fixe 20 %, calculée par la base
    category_id   INT NOT NULL REFERENCES categories(id) ON DELETE RESTRICT,
    active        BOOLEAN NOT NULL DEFAULT true,          -- false = archivé (déjà commandé, non supprimable)
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ------------------------------------------------------------
-- carts
-- ------------------------------------------------------------

CREATE TABLE carts (
    id          SERIAL PRIMARY KEY,
    reference   VARCHAR(10) NOT NULL UNIQUE,            -- format BSK-XXXXXX
    user_id     INT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    status      cart_status NOT NULL DEFAULT 'open',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Un seul panier "open" à la fois par utilisateur (les paniers "paid"
-- passés ne sont pas concernés par la contrainte)
CREATE UNIQUE INDEX idx_carts_open_unique_per_user
    ON carts (user_id)
    WHERE status = 'open';

-- ------------------------------------------------------------
-- cart_items (table de liaison carts <-> products)
-- ------------------------------------------------------------

CREATE TABLE cart_items (
    cart_id     INT NOT NULL REFERENCES carts(id) ON DELETE CASCADE,
    product_id  INT NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    quantity    INT NOT NULL CHECK (quantity >= 1),
    PRIMARY KEY (cart_id, product_id) -- un seul produit par panier ; on met à jour quantity
);

-- ------------------------------------------------------------
-- orders
-- ------------------------------------------------------------

CREATE TABLE orders (
    id             SERIAL PRIMARY KEY,
    reference      VARCHAR(10) NOT NULL UNIQUE,         -- format CMD-XXXXXX
    user_id        INT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    cart_id        INT REFERENCES carts(id) ON DELETE SET NULL, -- NULL si créée par un admin
    status         order_status NOT NULL DEFAULT 'pending',
    cancel_reason  TEXT,                                  -- rempli seulement si status = cancelled
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ------------------------------------------------------------
-- order_items (table de liaison orders <-> products, nom + prix figés)
-- ------------------------------------------------------------

CREATE TABLE order_items (
    id            SERIAL PRIMARY KEY,
    order_id      INT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id    INT NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    product_name  VARCHAR(255) NOT NULL,                 -- nom recopié au moment de la commande
    quantity      INT NOT NULL CHECK (quantity >= 1),
    price_ht      INTEGER NOT NULL CHECK (price_ht > 0), -- prix HT figé au moment de l'achat
    price_ttc     INTEGER GENERATED ALWAYS AS (ROUND(price_ht * 1.2)::INTEGER) STORED
);

-- ============================================================
-- Index sur les clés étrangères
-- (PostgreSQL n'indexe pas automatiquement les FK, seulement les
-- colonnes PRIMARY KEY / UNIQUE)
-- ============================================================

CREATE INDEX idx_sessions_user_id       ON sessions(user_id);
CREATE INDEX idx_products_category_id   ON products(category_id);
CREATE INDEX idx_carts_user_id          ON carts(user_id);
CREATE INDEX idx_cart_items_product_id  ON cart_items(product_id);
CREATE INDEX idx_orders_user_id         ON orders(user_id);
CREATE INDEX idx_orders_cart_id         ON orders(cart_id);
CREATE INDEX idx_order_items_order_id   ON order_items(order_id);
CREATE INDEX idx_order_items_product_id ON order_items(product_id);

-- ============================================================
-- Bonus (optionnel) : mise à jour automatique de updated_at
-- Décommentez si vous voulez que updated_at se mette à jour tout
-- seul à chaque UPDATE plutôt que de le faire à la main en Go.
-- ============================================================

-- CREATE OR REPLACE FUNCTION set_updated_at()
-- RETURNS TRIGGER AS $$
-- BEGIN
--     NEW.updated_at = now();
--     RETURN NEW;
-- END;
-- $$ LANGUAGE plpgsql;
--
-- CREATE TRIGGER trg_users_updated_at
--     BEFORE UPDATE ON users
--     FOR EACH ROW EXECUTE FUNCTION set_updated_at();
--
-- CREATE TRIGGER trg_products_updated_at
--     BEFORE UPDATE ON products
--     FOR EACH ROW EXECUTE FUNCTION set_updated_at();
--
-- CREATE TRIGGER trg_carts_updated_at
--     BEFORE UPDATE ON carts
--     FOR EACH ROW EXECUTE FUNCTION set_updated_at();
--
-- CREATE TRIGGER trg_orders_updated_at
--     BEFORE UPDATE ON orders
--     FOR EACH ROW EXECUTE FUNCTION set_updated_at();