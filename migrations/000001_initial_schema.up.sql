
-- TABLE APPLICATIONS
CREATE TABLE applications (
    id_application BIGINT GENERATED ALWAYS AS IDENTITY,
    nom VARCHAR(100) NOT NULL,
    api_key_hash VARCHAR(255) NOT NULL,
    statut BOOLEAN NOT NULL DEFAULT TRUE,
    date_creation TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT pk_applications
        PRIMARY KEY (id_application),

    CONSTRAINT uq_applications_api_key_hash
        UNIQUE (api_key_hash),

    CONSTRAINT uq_applications_nom
        UNIQUE (nom)
);


-- TABLE EMAILS
CREATE TABLE emails (
    id_email BIGINT GENERATED ALWAYS AS IDENTITY,
    id_application BIGINT NOT NULL,
    destinataire VARCHAR(255) NOT NULL,
    objet VARCHAR(255) NOT NULL,
    contenu TEXT NOT NULL,
    type_contenu VARCHAR(10) NOT NULL DEFAULT 'HTML',
    statut VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    date_creation TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    date_envoi TIMESTAMPTZ,
    message_erreur TEXT,

    CONSTRAINT pk_emails
        PRIMARY KEY (id_email),

    CONSTRAINT fk_emails_applications
        FOREIGN KEY (id_application)
        REFERENCES applications(id_application),

    CONSTRAINT chk_emails_statut
        CHECK (statut IN ('PENDING', 'SENDING', 'SENT', 'FAILED')),

    CONSTRAINT chk_emails_type_contenu
        CHECK (type_contenu IN ('TEXT', 'HTML'))
);


-- INDEX EMAILS
CREATE INDEX idx_emails_date_creation
    ON emails(date_creation);

CREATE INDEX idx_emails_id_application
    ON emails(id_application);

CREATE INDEX idx_emails_statut
    ON emails(statut);


-- TABLE PIECES_JOINTES
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


-- INDEX PIECES_JOINTES
CREATE INDEX idx_pieces_jointes_id_email
    ON pieces_jointes(id_email);
