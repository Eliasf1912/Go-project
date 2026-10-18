-- ============================================================
-- Boutique CLI — données de test (seed.sql, Phase 0)
-- À exécuter après init.sql
--
-- Contenu, tel que demandé pour la Phase 0 :
--   - un jeton de session de test, prêt à l'emploi, pour que B et C
--     puissent tester les routes protégées et /admin/* pendant que
--     l'authentification (Personne A) n'est pas encore terminée
--     (cf. règle d'équipe "bloqué par le travail d'un autre ? Utiliser
--     seed.sql ou un faux repository")
--   - une liste de départ de catégories
-- ============================================================

-- pgcrypto sert uniquement ici, dans le seed, à générer un hash
-- bcrypt sans dépendance externe. Le format produit ($2a$/$2b$) est
-- le même que celui de golang.org/x/crypto/bcrypt, donc ce compte
-- fonctionnera aussi avec /auth/login une fois cette route prête.
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ------------------------------------------------------------
-- Utilisateur de test (admin) + jeton de session prêt à l'emploi
-- ------------------------------------------------------------
-- email    : admin.test@boutique-cli.local
-- password : Test1234!
-- jeton    : dev-test-token-admin-0000000000
--
-- À utiliser directement dans les requêtes, sans passer par /auth/login :
--   Authorization: Bearer dev-test-token-admin-0000000000
--
-- Comme il s'agit d'un compte admin, ce jeton passe aussi bien
-- requireAuth (routes "Connecté") que requireAdmin (routes "Admin").

INSERT INTO users (email, password_hash, role, confirmed)
VALUES (
    'admin.test@boutique-cli.local',
    crypt('Test1234!', gen_salt('bf')),
    'admin',
    true
);

INSERT INTO sessions (token, user_id, expires_at)
SELECT 'dev-test-token-admin-0000000000', id, now() + interval '1 year'
FROM users
WHERE email = 'admin.test@boutique-cli.local';

-- ------------------------------------------------------------
-- Catégories de départ
-- ------------------------------------------------------------
-- Déjà en minuscules (contrainte CHECK (name = lower(name)) sur
-- categories.name)

INSERT INTO categories (name) VALUES
    ('informatique'),
    ('livres'),
    ('vêtements'),
    ('maison'),
    ('sport'),
    ('jouets'),
    ('beauté'),
    ('alimentation');

-- ------------------------------------------------------------
-- Produits de test (pour GET /products)
-- ------------------------------------------------------------
-- price_ht en centimes ; price_ttc est calculé par la base
-- (colonne générée, TVA fixe 20 %)

INSERT INTO products (reference, name, description, price_ht, category_id) VALUES
    ('PDT-A1B2C3', 'Clavier mécanique', 'Switches rouges, rétroéclairage RGB', 4999, (SELECT id FROM categories WHERE name = 'informatique')),
    ('PDT-D4E5F6', 'Souris sans fil', NULL, 1999, (SELECT id FROM categories WHERE name = 'informatique')),
    ('PDT-G7H8I9', 'Roman policier', 'Un thriller haletant', 833, (SELECT id FROM categories WHERE name = 'livres')),
    ('PDT-J1K2L3', 'T-shirt coton bio', NULL, 1500, (SELECT id FROM categories WHERE name = 'vêtements')),
    ('PDT-M4N5O6', 'Ballon de football', 'Taille 5, cuir synthétique', 2500, (SELECT id FROM categories WHERE name = 'sport')),
    ('PDT-P7Q8R9', 'Jeu de société', 'Pour 2 à 6 joueurs, dès 8 ans', 3333, (SELECT id FROM categories WHERE name = 'jouets'));