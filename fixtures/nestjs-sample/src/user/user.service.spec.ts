import { AuditService } from '../audit/audit.service';
import { UserRepository } from './user.repository';
import { UserService } from './user.service';

export async function userServiceDeleteTest(): Promise<void> {
  const service = new UserService(new UserRepository(), new AuditService());
  await service.deleteUser('u-1');
}
