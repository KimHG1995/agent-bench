import { UserService } from './user.service';

export class UserController {
  constructor(private readonly users: UserService) {}

  async deleteUser(id: string): Promise<void> {
    await this.users.deleteUser(id);
  }
}
