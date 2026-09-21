import { AuditService } from '../audit/audit.service';
import { UserRepository } from './user.repository';

export class UserService {
  constructor(
    private readonly users: UserRepository,
    private readonly audit: AuditService,
  ) {}

  async deleteUser(id: string): Promise<void> {
    await this.users.deleteById(id);
    this.audit.record('user.deleted', id);
  }
}
