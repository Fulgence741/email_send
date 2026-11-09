-- TABLE : APPLICATIONS

CREATE TABLE applications (
    id_application BIGINT GENERATED ALWAYS AS IDENTITY,

    nom VARCHAR(100) NOT NULL,
    api_key_hash VARCHAR(255) NOT NULL,

    statut VARCHAR(10) NOT NULL DEFAULT 'ACTIF',

    date_creation TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT pk_applications
        PRIMARY KEY (id_application),

    CONSTRAINT uq_applications_nom
        UNIQUE (nom),

    CONSTRAINT uq_applications_api_key_hash
        UNIQUE (api_key_hash),

    CONSTRAINT chk_applications_statut
        CHECK (statut IN ('ACTIF', 'INACTIF'))
);


-- TABLE : EMAILS

CREATE TABLE emails (
    id_email BIGINT GENERATED ALWAYS AS IDENTITY,

    id_application BIGINT NOT NULL,

    destinataire VARCHAR(255) NOT NULL,
    objet VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,

    type_contenu VARCHAR(10) NOT NULL DEFAULT 'HTML',

    statut VARCHAR(20) NOT NULL DEFAULT 'PENDING',

    -- Gestion de l'idempotence
    cle_idempotence VARCHAR(255) NOT NULL,

    -- Gestion des tentatives
    nombre_max_tentatives INTEGER NOT NULL DEFAULT 3,
    delai_retry_secondes INTEGER NOT NULL DEFAULT 60,

    -- Dates de traitement
    date_creation TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    date_debut_traitement TIMESTAMPTZ,
    date_fin_traitement TIMESTAMPTZ,
    date_prochaine_tentative TIMESTAMPTZ,

    message_erreur TEXT,

    CONSTRAINT pk_emails
        PRIMARY KEY (id_email),

    CONSTRAINT fk_emails_applications
        FOREIGN KEY (id_application)
        REFERENCES applications(id_application),

    -- Une même clé d'idempotence ne peut être utilisée
    -- qu'une seule fois par application
    CONSTRAINT uq_emails_idempotence
        UNIQUE (id_application, cle_idempotence),

    CONSTRAINT chk_emails_statut
        CHECK (
            statut IN (
                'PENDING',
                'SENDING',
                'SENT',
                'FAILED'
            )
        ),

    CONSTRAINT chk_emails_type_contenu
        CHECK (type_contenu IN ('TEXT', 'HTML')),

    CONSTRAINT chk_emails_nombre_max_tentatives
        CHECK (nombre_max_tentatives > 0),

    CONSTRAINT chk_emails_delai_retry
        CHECK (delai_retry_secondes >= 0),

    CONSTRAINT chk_emails_dates_traitement
        CHECK (
            date_fin_traitement IS NULL
            OR date_debut_traitement IS NULL
            OR date_fin_traitement >= date_debut_traitement
        )
);


-- INDEX : EMAILS

CREATE INDEX idx_emails_id_application
    ON emails(id_application);

CREATE INDEX idx_emails_statut
    ON emails(statut);

CREATE INDEX idx_emails_date_creation
    ON emails(date_creation);

CREATE INDEX idx_emails_date_prochaine_tentative
    ON emails(date_prochaine_tentative);


-- TABLE : PIECES_JOINTES

CREATE TABLE pieces_jointes (
    id_piece_jointe BIGINT GENERATED ALWAYS AS IDENTITY,

    id_email BIGINT NOT NULL,

    nom_fichier VARCHAR(255) NOT NULL,
    type_mime VARCHAR(150) NOT NULL,
    chemin_fichier TEXT NOT NULL,
    taille_fichier BIGINT NOT NULL,

    date_creation TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT pk_pieces_jointes
        PRIMARY KEY (id_piece_jointe),

    CONSTRAINT fk_pieces_jointes_emails
        FOREIGN KEY (id_email)
        REFERENCES emails(id_email)
        ON DELETE CASCADE,

    CONSTRAINT chk_pieces_jointes_taille
        CHECK (taille_fichier >= 0)
);


-- INDEX : PIECES_JOINTES

CREATE INDEX idx_pieces_jointes_id_email
    ON pieces_jointes(id_email);


-- TABLE : TENTATIVES_ENVOI

CREATE TABLE tentatives_envoi (
    id_tentative BIGINT GENERATED ALWAYS AS IDENTITY,

    id_email BIGINT NOT NULL,

    numero_tentative INTEGER NOT NULL,

    date_tentative TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    statut VARCHAR(20) NOT NULL DEFAULT 'EN_COURS',

    message_erreur TEXT,

    reponse_serveur TEXT,

    CONSTRAINT pk_tentatives_envoi
        PRIMARY KEY (id_tentative),

    CONSTRAINT fk_tentatives_envoi_emails
        FOREIGN KEY (id_email)
        REFERENCES emails(id_email)
        ON DELETE CASCADE,

    -- Un numéro de tentative ne peut pas être répété
    -- pour un même email
    CONSTRAINT uq_tentatives_email_numero
        UNIQUE (id_email, numero_tentative),

    CONSTRAINT chk_tentatives_numero
        CHECK (numero_tentative > 0),

    CONSTRAINT chk_tentatives_statut
        CHECK (
            statut IN (
                'EN_COURS',
                'REUSSIE',
                'ECHEC'
            )
        )
);


-- INDEX : TENTATIVES_ENVOI

CREATE INDEX idx_tentatives_envoi_id_email
    ON tentatives_envoi(id_email);

CREATE INDEX idx_tentatives_envoi_statut
    ON tentatives_envoi(statut);

CREATE INDEX idx_tentatives_envoi_date
    ON tentatives_envoi(date_tentative);
