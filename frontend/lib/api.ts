const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

export interface AuthResponse {
  success: boolean;
  message: string;
  token?: string;
  user?: {
    id: string;
    username: string;
    email: string;
    created_at: string;
    updated_at: string;
  };
}

export interface FileResponse {
  success: boolean;
  message: string;
  data?: any;
}

export interface FileItem {
  file_id: string;
  name: string;
  size: number;
  uploaded: string;
  status: string;
  chunks: number;
}

class APIClient {
  private getAuthHeader(): HeadersInit {
    if (typeof window === 'undefined') return {};
    const token = localStorage.getItem('authToken');
    return token ? { 'Authorization': `Bearer ${token}` } : {};
  }

  async register(username: string, email: string, password: string): Promise<AuthResponse> {
    const response = await fetch(`${API_URL}/api/v1/auth/register`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username, email, password }),
    });
    return response.json();
  }

  async login(email: string, password: string): Promise<AuthResponse> {
    const response = await fetch(`${API_URL}/api/v1/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password }),
    });
    return response.json();
  }

  async getProfile(): Promise<AuthResponse> {
    const response = await fetch(`${API_URL}/api/v1/auth/profile`, {
      headers: this.getAuthHeader(),
    });
    return response.json();
  }

  async uploadFile(file: File, onProgress?: (progress: number) => void): Promise<FileResponse> {
    const formData = new FormData();
    formData.append('file', file);

    const response = await fetch(`${API_URL}/api/v1/files/upload`, {
      method: 'POST',
      headers: this.getAuthHeader(),
      body: formData,
    });
    return response.json();
  }

  async listFiles(): Promise<FileResponse> {
    const response = await fetch(`${API_URL}/api/v1/files`, {
      headers: this.getAuthHeader(),
    });
    return response.json();
  }

  async downloadFile(fileId: string, fileName: string): Promise<void> {
    const response = await fetch(`${API_URL}/api/v1/files/${fileId}/download`, {
      headers: this.getAuthHeader(),
    });
    
    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.message || 'Download failed');
    }

    const blob = await response.blob();
    const url = window.URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = fileName;
    document.body.appendChild(a);
    a.click();
    window.URL.revokeObjectURL(url);
    document.body.removeChild(a);
  }

  async deleteFile(fileId: string): Promise<FileResponse> {
    const response = await fetch(`${API_URL}/api/v1/files/${fileId}`, {
      method: 'DELETE',
      headers: this.getAuthHeader(),
    });
    return response.json();
  }

  isAuthenticated(): boolean {
    if (typeof window === 'undefined') return false;
    return !!localStorage.getItem('authToken');
  }

  logout(): void {
    if (typeof window === 'undefined') return;
    localStorage.removeItem('authToken');
    localStorage.removeItem('currentUser');
  }

  saveAuth(token: string, user: any): void {
    if (typeof window === 'undefined') return;
    localStorage.setItem('authToken', token);
    localStorage.setItem('currentUser', JSON.stringify(user));
  }

  getCurrentUser(): any {
    if (typeof window === 'undefined') return null;
    const user = localStorage.getItem('currentUser');
    return user ? JSON.parse(user) : null;
  }
}

export const api = new APIClient();
