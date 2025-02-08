CREATE
DATABASE `leisu` /*!40100 DEFAULT CHARACTER SET utf8 */;
USE
`leisu`;

DROP TABLE IF EXISTS `article`;
CREATE TABLE `article`
(
    `id`           bigint(11) unsigned NOT NULL AUTO_INCREMENT,
    `title`        varchar(255) NOT NULL,
    `publish_time` char(32)     NOT NULL DEFAULT '',
    `content`      LONGTEXT     NOT NULL,
    `img`          varchar(255) NOT NULL DEFAULT '',
    `sport`        varchar(64)  NOT NULL DEFAULT '',
    `md5`          char(32)     NOT NULL DEFAULT '',
    `created_at`   timestamp NULL DEFAULT CURRENT_TIMESTAMP,
    `deleted_at`   timestamp NULL DEFAULT NULL,
    `updated_at`   timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `md5_idx` (`md5`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;