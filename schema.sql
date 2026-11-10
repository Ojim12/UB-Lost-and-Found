-- UB Lost and Found | CMPS2111 Phase 3 | MySQL 8.0.16+ (not yet run; test in your own environment)
CREATE DATABASE IF NOT EXISTS ub_lost_found CHARACTER SET utf8mb4;
USE ub_lost_found;

CREATE TABLE staff (
  staff_id CHAR(36) NOT NULL DEFAULT (UUID()),
  staff_name VARCHAR(100) NOT NULL,
  role VARCHAR(20) NOT NULL DEFAULT 'Office Staff',
  email VARCHAR(100) NOT NULL,
  password_hash VARCHAR(255) NOT NULL,
  PRIMARY KEY (staff_id),
  UNIQUE KEY uq_staff_email (email),
  CONSTRAINT ck_staff_role CHECK (role IN ('Admin','Office Staff')),
  CONSTRAINT ck_staff_email CHECK (email LIKE '%_@_%._%')
) ENGINE=InnoDB;

CREATE TABLE category (
  category_id CHAR(36) NOT NULL DEFAULT (UUID()),
  category_name VARCHAR(50) NOT NULL,
  PRIMARY KEY (category_id),
  UNIQUE KEY uq_category_name (category_name)
) ENGINE=InnoDB;

CREATE TABLE location (
  location_id CHAR(36) NOT NULL DEFAULT (UUID()),
  location_name VARCHAR(100) NOT NULL,
  PRIMARY KEY (location_id),
  UNIQUE KEY uq_location_name (location_name)
) ENGINE=InnoDB;

CREATE TABLE item (
  item_id CHAR(36) NOT NULL DEFAULT (UUID()),
  item_tag VARCHAR(20) NULL,
  description VARCHAR(500) NOT NULL,
  date_found DATE NOT NULL DEFAULT (CURDATE()),
  item_condition VARCHAR(20) NOT NULL DEFAULT 'Good',
  photo VARCHAR(255) NULL,
  finder_info VARCHAR(200) NULL,
  status VARCHAR(20) NOT NULL DEFAULT 'Unclaimed',
  category_id CHAR(36) NOT NULL,
  location_id CHAR(36) NOT NULL,
  logged_by_staff_id CHAR(36) NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  archived_at TIMESTAMP NULL DEFAULT NULL,
  PRIMARY KEY (item_id),
  UNIQUE KEY uq_item_tag (item_tag),
  KEY idx_item_status (status),
  KEY idx_item_search (category_id, location_id, date_found),
  CONSTRAINT ck_item_description CHECK (CHAR_LENGTH(description) >= 10),
  CONSTRAINT ck_item_condition CHECK (item_condition IN ('New','Good','Fair','Damaged')),
  CONSTRAINT ck_item_status CHECK (status IN ('Unclaimed','Pending Verification','Claimed')),
  CONSTRAINT ck_item_photo CHECK (photo IS NULL OR photo REGEXP '\\.(jpg|jpeg|png)$'),
  CONSTRAINT fk_item_category FOREIGN KEY (category_id) REFERENCES category (category_id) ON DELETE RESTRICT,
  CONSTRAINT fk_item_location FOREIGN KEY (location_id) REFERENCES location (location_id) ON DELETE RESTRICT,
  CONSTRAINT fk_item_staff FOREIGN KEY (logged_by_staff_id) REFERENCES staff (staff_id)
) ENGINE=InnoDB;

-- MySQL cannot use CURDATE() inside CHECK, so the no-future-date rule is a trigger
DELIMITER $$
CREATE TRIGGER trg_item_date_found BEFORE INSERT ON item
FOR EACH ROW
BEGIN
  IF NEW.date_found > CURDATE() THEN
    SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'date_found cannot be in the future';
  END IF;
END$$
DELIMITER ;

CREATE TABLE claimant (
  claimant_id CHAR(36) NOT NULL DEFAULT (UUID()),
  claimant_name VARCHAR(100) NOT NULL,
  claimant_type VARCHAR(20) NOT NULL DEFAULT 'Student',
  ub_id_number CHAR(10) NULL,
  contact_info VARCHAR(100) NOT NULL,
  PRIMARY KEY (claimant_id),
  CONSTRAINT ck_claimant_type CHECK (claimant_type IN ('Student','Faculty','Staff','Visitor')),
  CONSTRAINT ck_claimant_ub_id CHECK (ub_id_number IS NULL OR ub_id_number REGEXP '^[0-9]{10}$')
) ENGINE=InnoDB;

CREATE TABLE claim (
  claim_id CHAR(36) NOT NULL DEFAULT (UUID()),
  item_id CHAR(36) NOT NULL,
  claimant_id CHAR(36) NOT NULL,
  verified_by_staff_id CHAR(36) NULL,
  claim_status VARCHAR(10) NOT NULL DEFAULT 'Pending',
  proof_description VARCHAR(500) NOT NULL,
  verification_decision VARCHAR(255) NULL,
  claim_date TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  approved_item_id CHAR(36) GENERATED ALWAYS AS (IF(claim_status = 'Approved', item_id, NULL)) STORED,
  PRIMARY KEY (claim_id),
  UNIQUE KEY uq_one_approved_claim (approved_item_id),
  CONSTRAINT ck_claim_status CHECK (claim_status IN ('Pending','Approved','Rejected')),
  CONSTRAINT ck_claim_decided CHECK (claim_status = 'Pending' OR (verified_by_staff_id IS NOT NULL AND verification_decision IS NOT NULL)),
  CONSTRAINT fk_claim_item FOREIGN KEY (item_id) REFERENCES item (item_id),
  CONSTRAINT fk_claim_claimant FOREIGN KEY (claimant_id) REFERENCES claimant (claimant_id),
  CONSTRAINT fk_claim_staff FOREIGN KEY (verified_by_staff_id) REFERENCES staff (staff_id)
) ENGINE=InnoDB;

-- 1-year retention rule (run daily from a MySQL EVENT or a cron job)
UPDATE item
SET archived_at = NOW()
WHERE archived_at IS NULL
  AND status = 'Unclaimed'
  AND date_found < DATE_SUB(CURDATE(), INTERVAL 1 YEAR);

