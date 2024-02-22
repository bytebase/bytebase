/**
 * Hello, World!
 */
import { MigrationInterface, QueryRunner } from "typeorm";

export class PersonalAccessTokenFixExpirationTimeAutoUpdate1669892320740 implements MigrationInterface {
    public async up(queryRunner: QueryRunner): Promise<void> {
        await queryRunner.query(
            `ALTER TABLE d_b_personal_access_token CHANGE expirationTime expirationTime timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6)`,
        );
    }

    public async down(queryRunner: QueryRunner): Promise<void> {}
}
