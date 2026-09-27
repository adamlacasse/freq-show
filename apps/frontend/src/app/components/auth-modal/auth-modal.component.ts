import { Component, ElementRef, HostListener, ViewChild, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { HttpErrorResponse } from '@angular/common/http';
import { AuthService } from '../../services/auth.service';

@Component({
  selector: 'app-auth-modal',
  standalone: true,
  imports: [CommonModule, FormsModule],
  templateUrl: './auth-modal.component.html',
  styleUrl: './auth-modal.component.css'
})
export class AuthModalComponent {
  readonly authService = inject(AuthService);

  email = '';
  isSubmitting = signal<boolean>(false);
  errorMessage = signal<string | null>(null);
  submittedEmail = signal<string | null>(null);

  @ViewChild('emailInput') emailInput?: ElementRef<HTMLInputElement>;

  @HostListener('document:keydown.escape')
  onEscape(): void {
    if (this.authService.isModalOpen()) {
      this.close();
    }
  }

  close(): void {
    this.authService.closeModal();
    this.reset();
  }

  reset(): void {
    this.email = '';
    this.isSubmitting.set(false);
    this.errorMessage.set(null);
    this.submittedEmail.set(null);
  }

  onSubmit(): void {
    const trimmed = this.email.trim();
    if (!trimmed || this.isSubmitting()) {
      return;
    }

    this.isSubmitting.set(true);
    this.errorMessage.set(null);

    this.authService.requestMagicLink(trimmed).subscribe({
      next: () => {
        this.submittedEmail.set(trimmed);
        this.isSubmitting.set(false);
      },
      error: (err: HttpErrorResponse) => {
        this.isSubmitting.set(false);
        const serverError = err.error?.error;
        if (typeof serverError === 'string' && serverError.length > 0) {
          this.errorMessage.set(serverError);
        } else if (err.status === 429) {
          this.errorMessage.set('Too many requests. Please wait a moment before trying again.');
        } else if (err.status === 503) {
          this.errorMessage.set('Sign-in service is currently unavailable. Please try again later.');
        } else {
          this.errorMessage.set('Failed to send sign-in link. Please check your email and try again.');
        }
      }
    });
  }

  tryAnotherEmail(): void {
    this.reset();
    setTimeout(() => this.emailInput?.nativeElement?.focus(), 50);
  }
}
