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